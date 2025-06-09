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

package livestate

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"go.uber.org/zap"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
	"github.com/pipe-cd/community-plugins/plugins/azure-functions/provider"
)

type Plugin struct{}

func (p Plugin) GetLivestate(ctx context.Context, _ *sdk.ConfigNone, dts []*sdk.DeployTarget[config.AzureDeployTargetConfig], input *sdk.GetLivestateInput[config.AzureApplicationSpec]) (*sdk.GetLivestateResponse, error) {
	if len(dts) != 1 {
		return nil, fmt.Errorf("currently support only one deployment target, instead got %d", len(dts))
	}
	appCfg, err := input.Request.DeploymentSource.AppConfig()
	if err != nil {
		return nil, err
	}
	sdkClient, err := provider.NewAzureClient(ctx, dts[0].Config, map[string]*string{})
	if err != nil {
		return nil, err
	}
	switch appCfg.Spec.Kind {
	case config.FunctionKind:
		return getFunction(ctx, appCfg.Spec.FunctionManifest, sdkClient, input)
	default:
		return nil, fmt.Errorf("unknown app kind %s", appCfg.Spec.Kind)
	}
}

func getFunction(ctx context.Context, manifest *config.FunctionsSpec, client provider.AzureClient, input *sdk.GetLivestateInput[config.AzureApplicationSpec]) (*sdk.GetLivestateResponse, error) {
	liveState, err := getFunctionLiveState(ctx, manifest, client)
	if err != nil {
		input.Logger.Error("Failed to get function live state", zap.Error(err))
	}
	syncState, err := getFunctionSyncState(ctx, input.Request.DeploymentSource, manifest, client)
	if err != nil {
		input.Logger.Error("Failed to get function sync state", zap.Error(err))
	}
	resp := sdk.GetLivestateResponse{}
	if liveState != nil {
		resp.LiveState = *liveState
	}
	if syncState != nil {
		resp.SyncState = *syncState
	}
	return &resp, nil
}

func getFunctionLiveState(ctx context.Context, manifest *config.FunctionsSpec, c provider.AzureClient) (*sdk.ApplicationLiveState, error) {
	function, err := c.FunctionGetX(ctx, manifest.ResourceGroupName, manifest.FunctionName, "")
	if err != nil {
		return nil, err
	}
	resp := make([]sdk.ResourceState, 0)
	if function == nil {
		resp = append(resp, sdk.ResourceState{
			Name:              manifest.FunctionName,
			HealthStatus:      sdk.ResourceHealthStateUnhealthy,
			HealthDescription: "NotFound",
		})
		return &sdk.ApplicationLiveState{Resources: resp}, nil
	}
	functionHealthStatus := sdk.ResourceHealthStateUnknown
	functionHealthDescription := ""
	if function.Properties != nil && function.Properties.State != nil {
		if *function.Properties.State == "running" {
			functionHealthStatus = sdk.ResourceHealthStateHealthy
		} else {
			functionHealthStatus = sdk.ResourceHealthStateUnhealthy
			functionHealthDescription = *function.Properties.State
		}
	}
	deployTarget := ""
	if manifest.ArmTemplate != nil {
		deployTarget = manifest.ArmTemplate.DeploymentName
	}
	resp = append(resp, sdk.ResourceState{
		Name:              *function.Name,
		ID:                *function.ID,
		ResourceType:      "function",
		HealthStatus:      functionHealthStatus,
		HealthDescription: functionHealthDescription,
		DeployTarget:      deployTarget,
	})
	slots, err := c.FunctionListSlots(ctx, manifest.ResourceGroupName, manifest.FunctionName)
	if err != nil {
		return nil, err
	}
	for _, slot := range slots {
		slotHealthStatus := sdk.ResourceHealthStateUnknown
		slotHealthDescription := ""
		if slot.Properties != nil && slot.Properties.State != nil {
			if *slot.Properties.State == "running" {
				slotHealthStatus = sdk.ResourceHealthStateHealthy
			} else {
				slotHealthStatus = sdk.ResourceHealthStateUnhealthy
				slotHealthDescription = *slot.Properties.State
			}
		}
		resp = append(resp, sdk.ResourceState{
			Name:              *slot.Name,
			ID:                *slot.ID,
			ParentIDs:         []string{*function.ID},
			ResourceType:      "slot",
			HealthStatus:      slotHealthStatus,
			HealthDescription: slotHealthDescription,
			DeployTarget:      deployTarget,
		})
	}
	return &sdk.ApplicationLiveState{Resources: resp}, nil
}

func getFunctionSyncState(ctx context.Context, ds sdk.DeploymentSource[config.AzureApplicationSpec], manifest *config.FunctionsSpec, c provider.AzureClient) (*sdk.ApplicationSyncState, error) {
	if manifest == nil || manifest.ArmTemplate == nil {
		return &sdk.ApplicationSyncState{
			Status:      sdk.ApplicationSyncStateUnknown,
			ShortReason: "Sync state is not supported without template deployment",
		}, nil
	}
	diff, err := c.DeployARMTemplateWhatIf(ctx, manifest.ResourceGroupName, ds.ApplicationDirectory, *manifest.ArmTemplate)
	if err != nil {
		return nil, err
	}
	if len(diff.Changes)+len(diff.PotentialChanges) == 0 {
		return &sdk.ApplicationSyncState{
			Status: sdk.ApplicationSyncStateSynced,
		}, nil
	}
	commit := ds.CommitHash
	if len(commit) > 7 {
		commit = commit[:7]
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Diff between the defined state in Git at commit %s and actual live state:\n\n", commit))
	b.WriteString("--- Actual (LiveState)\n+++ Expected (Git)\n\n")
	add, del, update := 0, 0, 0
	for _, change := range diff.Changes {
		splits := strings.Split(*change.ResourceID, "/")
		resourceType := splits[len(splits)-2]
		resourceName := splits[len(splits)-1]
		switch *change.ChangeType {
		case armresources.ChangeTypeCreate:
			add++
			b.WriteString(fmt.Sprintf("+ %d. Name:%s Type:%s\n\n", add, resourceName, resourceType))
		case armresources.ChangeTypeDelete:
			del++
			b.WriteString(fmt.Sprintf("- %d. Name:%s Type:%s\n\n", del, resourceName, resourceType))
		case armresources.ChangeTypeModify:
			update++
			b.WriteString(fmt.Sprintf("# %d. Name:%s Type:%s\n\n", update, resourceName, resourceType))
			for _, delta := range change.Delta {
				renderWhatIfChange(b, *delta)
			}
			b.WriteString("\n")
		}
	}
	return &sdk.ApplicationSyncState{
		Status:      sdk.ApplicationSyncStateOutOfSync,
		ShortReason: fmt.Sprintf("There are %d resources need to add, %d resources need to delete, %d resource need to change", add, del, update),
		Reason:      b.String(),
	}, nil
}
func renderWhatIfChange(b strings.Builder, change armresources.WhatIfPropertyChange) {
	switch *change.PropertyChangeType {
	case armresources.PropertyChangeTypeCreate:
		b.WriteString(fmt.Sprintf("+ path:%s value:%s\n\n", *change.Path, change.After))
	case armresources.PropertyChangeTypeModify:
		b.WriteString(fmt.Sprintf("# path:%s value:%s -> %s\n\n", *change.Path, change.Before, change.After))
	case armresources.PropertyChangeTypeDelete:
		b.WriteString(fmt.Sprintf("- path:%s value:%s\n\n", *change.Path, change.Before))
	case armresources.PropertyChangeTypeArray:
		b.WriteString(fmt.Sprintf("# path:%s\n\n", *change.Path))
		for _, child := range change.Children {
			renderWhatIfChange(b, *child)
		}
	}
}
