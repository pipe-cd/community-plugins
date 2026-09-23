// Copyright 2026 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

const startTimeKey = "startTime"

// executeCheck polls the configured URL until it returns the expected
// status code, the timeout passes, or the stage is cancelled.
func (p *plugin) executeCheck(ctx context.Context, in *sdk.ExecuteStageInput[struct{}]) sdk.StageStatus {
	slp, err := in.Client.StageLogPersister()
	if err != nil {
		in.Logger.Error("No stage log persister available", zap.Error(err))
		return sdk.StageStatusFailure
	}
	opts, err := decode(in.Request.StageConfig)
	if err != nil {
		slp.Errorf("failed to decode the stage config: %v", err)
		return sdk.StageStatusFailure
	}

	// Retrieve the saved initialStart from the previous run.
	initialStart := p.retrieveStartTime(ctx, in.Client, in.Logger)
	if initialStart.IsZero() {
		// When this is the first run.
		initialStart = time.Now()
	}
	p.saveStartTime(ctx, in.Client, initialStart, in.Logger)

	client := &http.Client{Timeout: opts.Interval.Duration()}
	return check(ctx, client, opts, initialStart, slp)
}

func check(ctx context.Context, client *http.Client, opts HTTPCheckStageOptions, initialStart time.Time, slp sdk.StageLogPersister) sdk.StageStatus {
	// Count the budget from the first run so that a piped restart does not
	// reset the clock. The deadline also cuts off a probe that is still
	// waiting for a response when the budget ends.
	checkCtx, cancel := context.WithDeadline(ctx, initialStart.Add(opts.Timeout.Duration()))
	defer cancel()

	ticker := time.NewTicker(opts.Interval.Duration())
	defer ticker.Stop()

	slp.Infof("Checking %s for status %d every %v, timeout %v", opts.URL, opts.ExpectedCode, opts.Interval.Duration(), opts.Timeout.Duration())

	// The first probe uses ctx so that a stage restarted after its budget
	// passed still gets one attempt at a healthy endpoint.
	if ok := checkOnce(ctx, client, opts, slp); ok {
		return sdk.StageStatusSuccess
	}

	for {
		select {
		case <-ticker.C:
			if checkCtx.Err() != nil {
				// The budget ended while the previous probe was running.
				continue
			}
			if ok := checkOnce(checkCtx, client, opts, slp); ok {
				return sdk.StageStatusSuccess
			}

		case <-checkCtx.Done():
			if ctx.Err() != nil {
				slp.Info("HTTP check cancelled")
				// We can return any status here because the piped handles this case as cancelled by a user,
				// ignoring the result from a plugin.
				return sdk.StageStatusFailure
			}
			slp.Errorf("%s did not return status %d within %v", opts.URL, opts.ExpectedCode, opts.Timeout.Duration())
			return sdk.StageStatusFailure
		}
	}
}

func checkOnce(ctx context.Context, client *http.Client, opts HTTPCheckStageOptions, slp sdk.StageLogPersister) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, nil)
	if err != nil {
		slp.Errorf("failed to build request: %v", err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		slp.Infof("request failed: %v", err)
		return false
	}
	defer func() {
		// Drain the body so the connection can be reused by the next probe.
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != opts.ExpectedCode {
		slp.Infof("got status %d, want %d", resp.StatusCode, opts.ExpectedCode)
		return false
	}

	slp.Infof("%s returned status %d", opts.URL, resp.StatusCode)
	return true
}

func (p *plugin) retrieveStartTime(ctx context.Context, client *sdk.Client, logger *zap.Logger) time.Time {
	sec, ok, err := client.GetStageMetadata(ctx, startTimeKey)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get stage metadata %s", startTimeKey), zap.Error(err))
		return time.Time{}
	}
	if !ok {
		return time.Time{}
	}

	ut, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse stage metadata %s", startTimeKey), zap.Error(err))
		return time.Time{}
	}
	return time.Unix(ut, 0)
}

func (p *plugin) saveStartTime(ctx context.Context, client *sdk.Client, t time.Time, logger *zap.Logger) {
	value := strconv.FormatInt(t.Unix(), 10)
	if err := client.PutStageMetadata(ctx, startTimeKey, value); err != nil {
		logger.Error(fmt.Sprintf("failed to store %s as stage metadata %s", value, startTimeKey), zap.Error(err))
	}
}
