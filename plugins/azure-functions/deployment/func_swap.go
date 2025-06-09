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
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
	"github.com/pipe-cd/community-plugins/plugins/azure-functions/provider"
)

func (p *Plugin) executeAzureFuncSwapStage(ctx context.Context, dts []*sdk.DeployTarget[config.AzureDeployTargetConfig], input *sdk.ExecuteStageInput[config.AzureApplicationSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Start azure func swap")
	if len(dts) != 1 {
		lp.Errorf("Currently support only one deployment target, instead got %d", len(dts))
		return sdk.StageStatusFailure
	}
	sdkClient, err := provider.NewAzureClient(ctx, dts[0].Config, map[string]*string{
		provider.LabelManagedBy:   to.Ptr(provider.ManagedByPiped),
		provider.LabelPiped:       to.Ptr(input.Request.Deployment.PipedID),
		provider.LabelCommitHash:  to.Ptr(input.Request.TargetDeploymentSource.CommitHash),
		provider.LabelApplication: to.Ptr(input.Request.Deployment.ApplicationID),
	})
	if err != nil {
		lp.Errorf("Failed to create Azure Functions sdkClient: %v", err)
		return sdk.StageStatusFailure
	}
	appCfg, err := input.Request.TargetDeploymentSource.AppConfig()
	if err != nil {
		lp.Errorf("Failed to get AppConfig: %v", err)
		return sdk.StageStatusFailure
	}
	manifest := appCfg.Spec.FunctionManifest
	if manifest == nil {
		lp.Errorf("AppConfig.Spec.FunctionManifest is nil")
		return sdk.StageStatusFailure
	}
	stageConfigDump := input.Request.StageConfig
	var slotNames []string
	var stageConfig config.AzureFunctionSwapStageConfig
	if err = json.Unmarshal(stageConfigDump, &stageConfig); err != nil {
		lp.Errorf("cannot read sync stage config: %v", err)
	}
	if stageConfig.SlotName1 != "" {
		slotNames = append(slotNames, stageConfig.SlotName1)
	}
	if stageConfig.SlotName2 != "" {
		slotNames = append(slotNames, stageConfig.SlotName2)
	}
	if len(slotNames) == 0 {
		lp.Errorf("Swap template requires at least one slot set")
		return sdk.StageStatusFailure
	}
	needCreate, err := sdkClient.FunctionValidate(ctx, manifest.ResourceGroupName, manifest.FunctionName, slotNames)
	if err != nil {
		lp.Errorf("Failed to check manifest: %v", err)
		return sdk.StageStatusFailure
	}
	if needCreate {
		lp.Errorf("Sync stage should have resource existence guaranteed")
		return sdk.StageStatusFailure
	}
	err = sdkClient.FunctionSwapSlotX(ctx, manifest.ResourceGroupName, manifest.FunctionName, stageConfig.SlotName1, stageConfig.SlotName2)
	if err != nil {
		lp.Errorf("Failed to swap slots: %v", err)
		return sdk.StageStatusFailure
	}
	return sdk.StageStatusSuccess
}
