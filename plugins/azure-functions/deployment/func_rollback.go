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
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
	"github.com/pipe-cd/community-plugins/plugins/azure-functions/provider"
)

// currently rollback is implemented as a sync stage to the old configuration, which is similar to previously supported platform
// this would be updated later
func (p *Plugin) executeAzureFuncRollbackStage(ctx context.Context, dts []*sdk.DeployTarget[config.AzureDeployTargetConfig], input *sdk.ExecuteStageInput[config.AzureApplicationSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Start azure func rollback")
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
	appCfg, err := input.Request.RunningDeploymentSource.AppConfig()
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
	if len(stageConfigDump) > 0 {
		var stageConfig config.AzureFunctionSyncStageConfig
		if err = json.Unmarshal(stageConfigDump, &stageConfig); err != nil {
			lp.Errorf("cannot read sync stage config: %v", err)
		}
		slotNames = append(slotNames, stageConfig.SlotName)
	}
	if manifest.ArmTemplate != nil {
		lp.Infof("Start using arm template to deploy %s: template %s, parameter %s", manifest.ArmTemplate.DeploymentName, manifest.ArmTemplate.DeploymentTemplateFile, manifest.ArmTemplate.DeploymentParameterFile)
		err = sdkClient.DeployARMTemplate(ctx, manifest.ResourceGroupName, input.Request.TargetDeploymentSource.ApplicationDirectory, *manifest.ArmTemplate)
		if err != nil {
			lp.Errorf("Failed to deploy ARM template: %v", err)
			return sdk.StageStatusFailure
		}
	}
	needCreated, err := sdkClient.FunctionValidate(ctx, manifest.ResourceGroupName, manifest.FunctionName, slotNames)
	if err != nil {
		lp.Errorf("Failed to validate manifest: %v", err)
		return sdk.StageStatusFailure
	}
	if needCreated {
		lp.Errorf("Cannot find resource even after deploy template")
		return sdk.StageStatusFailure
	}
	var slotName string
	if len(slotNames) > 0 {
		slotName = slotNames[0]
	}
	current, err := sdkClient.FunctionGetX(ctx, manifest.ResourceGroupName, manifest.FunctionName, slotName)
	if err != nil {
		lp.Errorf("Failed to get Function %s: %v", manifest.FunctionName, err)
		return sdk.StageStatusFailure
	}

	if strings.Contains(*current.Kind, "linux") && *current.Properties.SKU == "Dynamic" { // Consumption Linux plan special treatment
		if err = sdkClient.FunctionRunFromPackageDeploymentX(ctx, manifest.ResourceGroupName, manifest.FunctionName, slotName, manifest.PackageUri); err != nil {
			lp.Errorf("Failed to deploy %s with 'WEBSITE_RUN_FROM_PACKAGE': %v", manifest.FunctionName, err)
			return sdk.StageStatusFailure
		}
	} else {
		if err = sdkClient.FunctionKuduDeploymentX(ctx, manifest.ResourceGroupName, manifest.FunctionName, slotName, manifest.PackageUri); err != nil {
			lp.Errorf("Failed to deploy %s with KuduDeployment: %v", manifest.FunctionName, err)
			return sdk.StageStatusFailure
		}
	}
	return sdk.StageStatusSuccess
}
