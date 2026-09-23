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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pipe-cd/piped-plugin-sdk-go/unit"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  string
		want    HTTPCheckStageOptions
		wantErr bool
	}{
		{
			name:   "defaults applied",
			config: `{"url": "http://localhost:8080/healthz"}`,
			want: HTTPCheckStageOptions{
				URL:          "http://localhost:8080/healthz",
				ExpectedCode: 200,
				Timeout:      unit.Duration(time.Minute),
				Interval:     unit.Duration(5 * time.Second),
			},
		},
		{
			name:   "all fields set",
			config: `{"url": "https://example.com/ready", "expectedCode": 204, "timeout": "30s", "interval": "2s"}`,
			want: HTTPCheckStageOptions{
				URL:          "https://example.com/ready",
				ExpectedCode: 204,
				Timeout:      unit.Duration(30 * time.Second),
				Interval:     unit.Duration(2 * time.Second),
			},
		},
		{
			name:    "missing url",
			config:  `{}`,
			wantErr: true,
		},
		{
			name:    "bad scheme",
			config:  `{"url": "ftp://example.com"}`,
			wantErr: true,
		},
		{
			name:    "missing host",
			config:  `{"url": "http://"}`,
			wantErr: true,
		},
		{
			name:    "bad status code",
			config:  `{"url": "http://example.com", "expectedCode": 42}`,
			wantErr: true,
		},
		{
			name:    "interval not less than timeout",
			config:  `{"url": "http://example.com", "timeout": "5s", "interval": "5s"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decode(json.RawMessage(tt.config))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
