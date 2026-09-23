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
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/creasty/defaults"
	"github.com/pipe-cd/piped-plugin-sdk-go/unit"
)

// HTTPCheckStageOptions contains configurable values for an HTTP_CHECK stage.
type HTTPCheckStageOptions struct {
	// URL is the endpoint to check.
	URL string `json:"url"`
	// ExpectedCode is the HTTP status code treated as healthy.
	ExpectedCode int `json:"expectedCode,omitempty" default:"200"`
	// Timeout is how long to keep checking before failing the stage.
	Timeout unit.Duration `json:"timeout,omitempty" default:"1m"`
	// Interval is how long to wait between checks.
	Interval unit.Duration `json:"interval,omitempty" default:"5s"`
}

func (o HTTPCheckStageOptions) validate() error {
	if o.URL == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(o.URL)
	if err != nil {
		return fmt.Errorf("url is invalid: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("url host is required")
	}
	if o.ExpectedCode < 100 || o.ExpectedCode > 599 {
		return fmt.Errorf("expectedCode must be a valid HTTP status code")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	if o.Interval <= 0 {
		return fmt.Errorf("interval must be greater than 0")
	}
	if o.Interval >= o.Timeout {
		return fmt.Errorf("interval must be less than timeout")
	}
	return nil
}

// decode decodes the raw JSON data and validates it.
func decode(data json.RawMessage) (HTTPCheckStageOptions, error) {
	var opts HTTPCheckStageOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return HTTPCheckStageOptions{}, fmt.Errorf("failed to unmarshal the config: %w", err)
	}
	if err := defaults.Set(&opts); err != nil {
		return HTTPCheckStageOptions{}, fmt.Errorf("failed to set default values for stage config: %w", err)
	}
	if err := opts.validate(); err != nil {
		return HTTPCheckStageOptions{}, fmt.Errorf("failed to validate the config: %w", err)
	}
	return opts, nil
}
