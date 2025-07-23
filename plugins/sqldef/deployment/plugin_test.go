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

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/stretchr/testify/assert"
)

func Test_BuildPipelineSyncStages(t *testing.T) {
	tests := []struct {
		name     string
		input    *sdk.BuildPipelineSyncStagesInput
		expected *sdk.BuildPipelineSyncStagesResponse
	}{
		{
			name: "rollback = false",
			input: &sdk.BuildPipelineSyncStagesInput{
				Request: sdk.BuildPipelineSyncStagesRequest{
					Stages: []sdk.StageConfig{
						{Name: sqldefStagePlan, Index: 1},
						{Name: sqldefStageApply, Index: 2},
					},
					Rollback: false,
				},
			},
			expected: &sdk.BuildPipelineSyncStagesResponse{
				Stages: []sdk.PipelineStage{
					{Name: sqldefStagePlan, Index: 1, Rollback: false, Metadata: map[string]string{}, AvailableOperation: sdk.ManualOperationNone},
					{Name: sqldefStageApply, Index: 2, Rollback: false, Metadata: map[string]string{}, AvailableOperation: sdk.ManualOperationNone},
				},
			},
		},
		{
			name: "rollback = true",
			input: &sdk.BuildPipelineSyncStagesInput{
				Request: sdk.BuildPipelineSyncStagesRequest{
					Stages: []sdk.StageConfig{
						{Name: sqldefStagePlan, Index: 4},
						{Name: sqldefStageApply, Index: 3},
					},
					Rollback: true,
				},
			},
			expected: &sdk.BuildPipelineSyncStagesResponse{
				Stages: []sdk.PipelineStage{
					{Name: sqldefStagePlan, Index: 4, Rollback: false, Metadata: map[string]string{}, AvailableOperation: sdk.ManualOperationNone},
					{Name: sqldefStageApply, Index: 3, Rollback: false, Metadata: map[string]string{}, AvailableOperation: sdk.ManualOperationNone},
					{Name: sqldefStageRollback, Index: 3, Rollback: true, Metadata: map[string]string{}, AvailableOperation: sdk.ManualOperationNone},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Plugin{}
			got, err := p.BuildPipelineSyncStages(t.Context(), nil, tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}

}

func TestPlugin_BuildQuickSyncStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *sdk.BuildQuickSyncStagesInput
		expected *sdk.BuildQuickSyncStagesResponse
	}{
		{
			name: "rollback = false",
			input: &sdk.BuildQuickSyncStagesInput{
				Request: sdk.BuildQuickSyncStagesRequest{
					Rollback: false,
				},
			},
			expected: &sdk.BuildQuickSyncStagesResponse{
				Stages: []sdk.QuickSyncStage{
					{
						Name:               sqldefStageApply,
						Description:        "Apply changes to target DB",
						Rollback:           false,
						Metadata:           map[string]string{},
						AvailableOperation: sdk.ManualOperationNone,
					},
				},
			},
		},
		{
			name: "rollback = true",
			input: &sdk.BuildQuickSyncStagesInput{
				Request: sdk.BuildQuickSyncStagesRequest{
					Rollback: true,
				},
			},
			expected: &sdk.BuildQuickSyncStagesResponse{
				Stages: []sdk.QuickSyncStage{
					{
						Name:               sqldefStageApply,
						Description:        "Apply changes to target DB",
						Rollback:           false,
						Metadata:           map[string]string{},
						AvailableOperation: sdk.ManualOperationNone,
					},
					{
						Name:               sqldefStageRollback,
						Description:        "Rollback to previous DB schema",
						Rollback:           true,
						Metadata:           map[string]string{},
						AvailableOperation: sdk.ManualOperationNone,
					},
				},
			},
		},
	}

	p := &Plugin{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := p.BuildQuickSyncStages(t.Context(), nil, tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
