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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
	"github.com/pipe-cd/piped-plugin-sdk-go/unit"
)

func testOptions(url string) HTTPCheckStageOptions {
	return HTTPCheckStageOptions{
		URL:          url,
		ExpectedCode: 200,
		Timeout:      unit.Duration(500 * time.Millisecond),
		Interval:     unit.Duration(20 * time.Millisecond),
	}
}

func TestCheck_ImmediateSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status := check(context.Background(), server.Client(), testOptions(server.URL), time.Now(), logpersistertest.NewTestLogPersister(t))
	assert.Equal(t, sdk.StageStatusSuccess, status)
}

func TestCheck_EventualSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status := check(context.Background(), server.Client(), testOptions(server.URL), time.Now(), logpersistertest.NewTestLogPersister(t))
	assert.Equal(t, sdk.StageStatusSuccess, status)
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

func TestCheck_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	status := check(context.Background(), server.Client(), testOptions(server.URL), time.Now(), logpersistertest.NewTestLogPersister(t))
	assert.Equal(t, sdk.StageStatusFailure, status)
}

func TestCheck_RestartAfterTimeoutHealthy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// The stage restarted after its budget already passed. The healthy
	// endpoint should still pass on the immediate probe.
	initialStart := time.Now().Add(-time.Second)
	status := check(context.Background(), server.Client(), testOptions(server.URL), initialStart, logpersistertest.NewTestLogPersister(t))
	assert.Equal(t, sdk.StageStatusSuccess, status)
}

func TestCheck_RestartAfterTimeoutUnhealthy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	initialStart := time.Now().Add(-time.Second)
	status := check(context.Background(), server.Client(), testOptions(server.URL), initialStart, logpersistertest.NewTestLogPersister(t))
	assert.Equal(t, sdk.StageStatusFailure, status)
}

func TestCheck_TimeoutWhileProbeInFlight(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Answer well after the budget has passed.
		select {
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	start := time.Now()
	status := check(context.Background(), server.Client(), testOptions(server.URL), start, logpersistertest.NewTestLogPersister(t))

	assert.Equal(t, sdk.StageStatusFailure, status)
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestCheck_Cancel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	opts := testOptions(server.URL)
	opts.Timeout = unit.Duration(10 * time.Second)

	resultCh := make(chan sdk.StageStatus, 1)
	go func() {
		resultCh <- check(ctx, server.Client(), opts, time.Now(), logpersistertest.NewTestLogPersister(t))
	}()

	cancel()

	select {
	case status := <-resultCh:
		assert.Equal(t, sdk.StageStatusFailure, status)
	case <-time.After(2 * time.Second):
		t.Error("check() did not end after cancellation")
	}
}
