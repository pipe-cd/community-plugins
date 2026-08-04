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
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

func TestPlugin_FetchDefinedStages(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	got := p.FetchDefinedStages()

	assert.Equal(t, []string{
		string(AzureFuncSync),
		string(AzureFuncSwap),
		string(AzureFuncRollback),
	}, got)
}

func TestPlugin_BuildPipelineSyncStages(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	input := &sdk.BuildPipelineSyncStagesInput{
		Request: sdk.BuildPipelineSyncStagesRequest{
			Rollback: true,
			Stages: []sdk.StageConfig{
				{Name: string(AzureFuncSync), Index: 0},
			},
		},
	}

	got, err := p.BuildPipelineSyncStages(context.Background(), nil, input)

	assert.NoError(t, err)
	assert.Equal(t, buildPipeline(input.Request.Stages, input.Request.Rollback), got.Stages)
}

func TestPlugin_BuildQuickSyncStages(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	input := &sdk.BuildQuickSyncStagesInput{
		Request: sdk.BuildQuickSyncStagesRequest{Rollback: true},
	}

	got, err := p.BuildQuickSyncStages(context.Background(), nil, input)

	assert.NoError(t, err)
	assert.Equal(t, buildQuickSync(true), got.Stages)
}

func TestPlugin_DetermineStrategy(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	input := &sdk.DetermineStrategyInput[config.AzureApplicationSpec]{}

	got, err := p.DetermineStrategy(context.Background(), nil, input)

	assert.NoError(t, err)
	assert.Equal(t, sdk.SyncStrategyQuickSync, got.Strategy)
}

func TestPlugin_DetermineVersions(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		deployConfig *sdk.ApplicationConfig[config.AzureApplicationSpec]
		expectErr    bool
		expectPanic  bool
		expected     []sdk.ArtifactVersion
	}{
		{
			name: "function kind returns the package uri as the version",
			deployConfig: &sdk.ApplicationConfig[config.AzureApplicationSpec]{
				Spec: &config.AzureApplicationSpec{
					Kind: config.FunctionKind,
					FunctionManifest: &config.FunctionsSpec{
						PackageUri: "https://example.blob.core.windows.net/pkg/app.zip",
					},
				},
			},
			expected: []sdk.ArtifactVersion{
				{
					Kind:    ArtifactAzureBlob,
					Name:    "packageUri",
					Version: "https://example.blob.core.windows.net/pkg/app.zip",
					URL:     "https://example.blob.core.windows.net/pkg/app.zip",
				},
			},
		},
		{
			name:         "missing application config returns an error",
			deployConfig: nil,
			expectErr:    true,
		},
		{
			name: "unsupported kind panics",
			deployConfig: &sdk.ApplicationConfig[config.AzureApplicationSpec]{
				Spec: &config.AzureApplicationSpec{Kind: "unsupported"},
			},
			expectPanic: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := &Plugin{}
			input := &sdk.DetermineVersionsInput[config.AzureApplicationSpec]{
				Request: sdk.DetermineVersionsRequest[config.AzureApplicationSpec]{
					DeploymentSource: sdk.DeploymentSource[config.AzureApplicationSpec]{
						ApplicationConfig: tc.deployConfig,
					},
				},
			}

			if tc.expectPanic {
				assert.Panics(t, func() {
					_, _ = p.DetermineVersions(context.Background(), nil, input) //nolint:errcheck
				})
				return
			}

			got, err := p.DetermineVersions(context.Background(), nil, input)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got.Versions)
		})
	}
}

// newExecuteStageInput builds a minimal ExecuteStageInput usable for testing
// the early-exit validation branches of ExecuteStage that don't reach out to
// Azure.
func newExecuteStageInput(t *testing.T, stageName string) *sdk.ExecuteStageInput[config.AzureApplicationSpec] {
	t.Helper()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	return &sdk.ExecuteStageInput[config.AzureApplicationSpec]{
		Request: sdk.ExecuteStageRequest[config.AzureApplicationSpec]{
			StageName: stageName,
		},
		Client: sdk.NewClient(nil, "azure-functions", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}
}

func TestPlugin_ExecuteStage_InvalidDeployTargetCount(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		stageName string
		dts       []*sdk.DeployTarget[config.AzureDeployTargetConfig]
	}{
		{
			name:      "sync stage with no deploy targets",
			stageName: string(AzureFuncSync),
			dts:       []*sdk.DeployTarget[config.AzureDeployTargetConfig]{},
		},
		{
			name:      "sync stage with multiple deploy targets",
			stageName: string(AzureFuncSync),
			dts: []*sdk.DeployTarget[config.AzureDeployTargetConfig]{
				{Name: "dt-1"},
				{Name: "dt-2"},
			},
		},
		{
			name:      "swap stage with no deploy targets",
			stageName: string(AzureFuncSwap),
			dts:       []*sdk.DeployTarget[config.AzureDeployTargetConfig]{},
		},
		{
			name:      "rollback stage with no deploy targets",
			stageName: string(AzureFuncRollback),
			dts:       []*sdk.DeployTarget[config.AzureDeployTargetConfig]{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := &Plugin{}
			input := newExecuteStageInput(t, tc.stageName)

			got, err := p.ExecuteStage(context.Background(), nil, tc.dts, input)

			assert.NoError(t, err)
			assert.Equal(t, sdk.StageStatusFailure, got.Status)
		})
	}
}

func TestPlugin_ExecuteStage_UnknownStagePanics(t *testing.T) {
	t.Parallel()

	p := &Plugin{}
	input := newExecuteStageInput(t, "UNKNOWN_STAGE")

	assert.Panics(t, func() {
		_, _ = p.ExecuteStage(context.Background(), nil, nil, input) //nolint:errcheck
	})
}
