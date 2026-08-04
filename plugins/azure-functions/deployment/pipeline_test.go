// Copyright 2025 The PipeCD Authors.
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

package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func TestBuildQuickSync(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		autoRollback bool
		expected     []sdk.QuickSyncStage
	}{
		{
			name:         "without auto rollback returns only the sync stage",
			autoRollback: false,
			expected: []sdk.QuickSyncStage{
				{
					Name:               string(AzureFuncSync),
					Description:        "",
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
		{
			name:         "with auto rollback appends a rollback stage",
			autoRollback: true,
			expected: []sdk.QuickSyncStage{
				{
					Name:               string(AzureFuncSync),
					Description:        "",
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncRollback),
					Description:        "",
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
					Rollback:           true,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildQuickSync(tc.autoRollback)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestBuildPipeline(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		stages       []sdk.StageConfig
		autoRollback bool
		expected     []sdk.PipelineStage
	}{
		{
			name:         "empty stages without auto rollback returns empty pipeline",
			stages:       []sdk.StageConfig{},
			autoRollback: false,
			expected:     []sdk.PipelineStage{},
		},
		{
			name: "single stage without auto rollback",
			stages: []sdk.StageConfig{
				{Name: string(AzureFuncSync), Index: 0},
			},
			autoRollback: false,
			expected: []sdk.PipelineStage{
				{
					Name:               string(AzureFuncSync),
					Index:              0,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
		{
			name: "multiple stages with auto rollback appends rollback at minimum index",
			stages: []sdk.StageConfig{
				{Name: string(AzureFuncSync), Index: 1},
				{Name: string(AzureFuncSwap), Index: 2},
			},
			autoRollback: true,
			expected: []sdk.PipelineStage{
				{
					Name:               string(AzureFuncSync),
					Index:              1,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncSwap),
					Index:              2,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncRollback),
					Index:              1,
					Rollback:           true,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
		{
			name: "rollback stage index follows minimum index even when stages are out of order",
			stages: []sdk.StageConfig{
				{Name: string(AzureFuncSwap), Index: 5},
				{Name: string(AzureFuncSync), Index: 2},
				{Name: string(AzureFuncRollback), Index: 8},
			},
			autoRollback: true,
			expected: []sdk.PipelineStage{
				{
					Name:               string(AzureFuncSwap),
					Index:              5,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncSync),
					Index:              2,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncRollback),
					Index:              8,
					Rollback:           false,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
				{
					Name:               string(AzureFuncRollback),
					Index:              2,
					Rollback:           true,
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildPipeline(tc.stages, tc.autoRollback)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestBuildPipeline_EmptyStagesWithAutoRollback documents the current edge-case
// behavior: computing the rollback stage index relies on finding the minimum
// index among the given stages, which panics when there are no stages to
// rollback to.
func TestBuildPipeline_EmptyStagesWithAutoRollback(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		buildPipeline([]sdk.StageConfig{}, true)
	})
}
