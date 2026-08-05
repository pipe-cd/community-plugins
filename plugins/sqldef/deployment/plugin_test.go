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
	"context"
	"path/filepath"
	"testing"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
)

func TestPlugin_FetchDefinedStages(t *testing.T) {
	t.Parallel()

	p := &Plugin{}

	got := p.FetchDefinedStages()

	assert.Equal(t, []string{sqldefStagePlan, sqldefStageApply, sqldefStageRollback}, got)
}

func TestPlugin_DetermineVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		commitHash string
	}{
		{name: "non-empty commit hash", commitHash: "0123456789"},
		{name: "empty commit hash", commitHash: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &Plugin{}
			input := &sdk.DetermineVersionsInput[config.ApplicationConfigSpec]{
				Request: sdk.DetermineVersionsRequest[config.ApplicationConfigSpec]{
					DeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
						CommitHash: tt.commitHash,
					},
				},
			}

			got, err := p.DetermineVersions(t.Context(), nil, input)

			assert.NoError(t, err)
			assert.Equal(t, &sdk.DetermineVersionsResponse{
				Versions: []sdk.ArtifactVersion{{Version: tt.commitHash}},
			}, got)
		})
	}
}

func TestPlugin_DetermineStrategy(t *testing.T) {
	t.Parallel()

	p := &Plugin{}

	got, err := p.DetermineStrategy(t.Context(), nil, &sdk.DetermineStrategyInput[config.ApplicationConfigSpec]{})

	assert.NoError(t, err)
	assert.Nil(t, got)
}

// Note: ExecuteStage's behavior for an unrecognized stage name (currently a
// panic) is covered by a separate fix that replaces the panic with an error
// return; that PR owns the regression test for this branch, since asserting
// the panic here would break as soon as that fix lands.

func TestPlugin_ExecuteStage_Dispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stageName string
		dryRun    bool
	}{
		{name: "plan stage", stageName: sqldefStagePlan, dryRun: true},
		{name: "apply stage", stageName: sqldefStageApply, dryRun: false},
		{name: "rollback stage", stageName: sqldefStageRollback, dryRun: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			testRegistry := toolregistrytest.NewTestToolRegistry(t)
			mockSqldef := &MockSqldefProvider{}

			mockSqldef.On("Init",
				mock.AnythingOfType("logpersistertest.TestLogPersister"),
				"testuser", "testpass", "localhost", "3306", "testdb",
				filepath.Join("testdata", "schema.sql"),
				mock.AnythingOfType("string"),
			).Return()
			mockSqldef.On("Execute", ctx, tt.dryRun).Return(nil)

			input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
				Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
					StageName:   tt.stageName,
					StageConfig: []byte(``),
					RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
						ApplicationDirectory: "testdata",
						CommitHash:           "0123456789",
					},
					TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
						ApplicationDirectory: "testdata",
						CommitHash:           "0123456789",
					},
					Deployment: sdk.Deployment{
						PipedID:       "piped-id",
						ApplicationID: "app-id",
					},
				},
				Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
			}

			deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
				{
					Name: "test-mysql",
					Config: config.DeployTargetConfig{
						DbType:   config.DBTypeMySQL,
						Username: "testuser",
						Password: "testpass",
						Host:     "localhost",
						Port:     "3306",
						DBName:   "testdb",
					},
				},
			}

			plugin := createPluginWithMockSqldef(mockSqldef)

			got, err := plugin.ExecuteStage(ctx, nil, deployTargets, input)

			assert.NoError(t, err)
			assert.Equal(t, &sdk.ExecuteStageResponse{Status: sdk.StageStatusSuccess}, got)
			mockSqldef.AssertExpectations(t)
		})
	}
}

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
