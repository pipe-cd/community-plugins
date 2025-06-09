// Copyright \d{4} The PipeCD Authors.
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

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

// Plugin implements the sdk.DeploymentPlugin interface.
type Plugin struct{}

func (p *Plugin) FetchDefinedStages() []string {
	return allStages
}

func (p *Plugin) BuildPipelineSyncStages(ctx context.Context, _ *sdk.ConfigNone, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: buildPipeline(input.Request.Stages, input.Request.Rollback),
	}, nil
}

func (p *Plugin) ExecuteStage(ctx context.Context, _ *sdk.ConfigNone, dts []*sdk.DeployTarget[config.AzureDeployTargetConfig], input *sdk.ExecuteStageInput[config.AzureApplicationSpec]) (*sdk.ExecuteStageResponse, error) {
	switch input.Request.StageName {
	case string(AzureFuncSync):
		return &sdk.ExecuteStageResponse{
			Status: p.executeAzureFuncSyncStage(ctx, dts, input),
		}, nil
	case string(AzureFuncSwap):
		return &sdk.ExecuteStageResponse{
			Status: p.executeAzureFuncSwapStage(ctx, dts, input),
		}, nil
	case string(AzureFuncRollback):
		return &sdk.ExecuteStageResponse{
			Status: p.executeAzureFuncRollbackStage(ctx, dts, input),
		}, nil
	default:
		panic("unimplemented stage: " + input.Request.StageName)
	}
}

const (
	ArtifactAzureBlob sdk.ArtifactKind = 5
)

func (p *Plugin) DetermineVersions(ctx context.Context, _ *sdk.ConfigNone, d *sdk.DetermineVersionsInput[config.AzureApplicationSpec]) (*sdk.DetermineVersionsResponse, error) {
	appCfg, err := d.Request.DeploymentSource.AppConfig()
	if err != nil {
		return nil, err
	}
	switch appCfg.Spec.Kind {
	case config.FunctionKind:
		return &sdk.DetermineVersionsResponse{
			Versions: []sdk.ArtifactVersion{
				{
					Kind:    ArtifactAzureBlob,
					Name:    "packageUri",
					Version: appCfg.Spec.FunctionManifest.PackageUri,
					URL:     appCfg.Spec.FunctionManifest.PackageUri,
				},
			},
		}, nil
	default:
		panic("not supported kind: " + appCfg.Spec.Kind)
	}
}

func (p *Plugin) DetermineStrategy(ctx context.Context, _ *sdk.ConfigNone, d *sdk.DetermineStrategyInput[config.AzureApplicationSpec]) (*sdk.DetermineStrategyResponse, error) {
	// return quicksync here is fine, pipelines one would require pipelines set in application config file (app.pipecd.yaml)
	return &sdk.DetermineStrategyResponse{
		Strategy: sdk.SyncStrategyQuickSync,
	}, nil
}

func (p *Plugin) BuildQuickSyncStages(ctx context.Context, config *sdk.ConfigNone, input *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	return &sdk.BuildQuickSyncStagesResponse{
		Stages: buildQuickSync(input.Request.Rollback),
	}, nil
}
