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
	"errors"
	"slices"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/opentofu/config"
)

const (
	// OPENTOFU_PLAN stage executes `tofu plan`.
	stagePlan = "OPENTOFU_PLAN"
	// OPENTOFU_APPLY stage executes `tofu apply`.
	stageApply = "OPENTOFU_APPLY"
	// OPENTOFU_ROLLBACK stage rollbacks by executing 'tofu apply' for the previous state.
	stageRollback = "OPENTOFU_ROLLBACK"
)

// Plugin implements sdk.DeploymentPlugin for OpenTofu.
type Plugin struct{}

var _ sdk.DeploymentPlugin[config.Config, config.DeployTargetConfig, config.ApplicationConfigSpec] = (*Plugin)(nil)

func (*Plugin) FetchDefinedStages() []string {
	return []string{
		stagePlan,
		stageApply,
		stageRollback,
	}
}

// BuildPipelineSyncStages builds the stages that will be executed by the plugin.
func (p *Plugin) BuildPipelineSyncStages(ctx context.Context, cfg *config.Config, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	reqStages := input.Request.Stages
	out := make([]sdk.PipelineStage, 0, len(reqStages))

	for _, s := range reqStages {
		out = append(out, sdk.PipelineStage{
			Index:              s.Index,
			Name:               s.Name,
			Rollback:           false,
			Metadata:           make(map[string]string),
			AvailableOperation: sdk.ManualOperationNone,
		})
	}
	if input.Request.Rollback {
		// minIndex from reqStages to ensure the rollback stage executes first.
		minIndex := slices.MinFunc(reqStages, func(a, b sdk.StageConfig) int { return a.Index - b.Index }).Index
		out = append(out, sdk.PipelineStage{
			Name:               stageRollback,
			Index:              minIndex,
			Rollback:           true,
			Metadata:           make(map[string]string),
			AvailableOperation: sdk.ManualOperationNone,
		})
	}

	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: out,
	}, nil
}

// ExecuteStage executes the given stage.
func (p *Plugin) ExecuteStage(ctx context.Context, cfg *config.Config, dts []*sdk.DeployTarget[config.DeployTargetConfig], input *sdk.ExecuteStageInput[config.ApplicationConfigSpec]) (*sdk.ExecuteStageResponse, error) {
	switch input.Request.StageName {
	case stagePlan:
		return &sdk.ExecuteStageResponse{
			Status: p.executePlanStage(ctx, input, dts),
		}, nil
	case stageApply:
		panic("unimplemented")
	case stageRollback:
		panic("unimplemented")
	default:
		return nil, errors.New("unsupported stage")
	}
}

// DetermineVersions determines the versions of artifacts for the deployment.
func (p *Plugin) DetermineVersions(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.DetermineVersionsInput[config.ApplicationConfigSpec],
) (*sdk.DetermineVersionsResponse, error) {
	// TODO: Implement DetermineVersions logic
	return &sdk.DetermineVersionsResponse{
		Versions: []sdk.ArtifactVersion{},
	}, nil
}

// DetermineStrategy determines the strategy for the deployment.
func (p *Plugin) DetermineStrategy(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.DetermineStrategyInput[config.ApplicationConfigSpec],
) (*sdk.DetermineStrategyResponse, error) {
	// TODO: Implement DetermineStrategy logic
	return nil, nil
}

// BuildQuickSyncStages builds the stages for quick sync.
func (p *Plugin) BuildQuickSyncStages(ctx context.Context, cfg *config.Config, input *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	stages := make([]sdk.QuickSyncStage, 0, 2)
	stages = append(stages, sdk.QuickSyncStage{
		Name:               stageApply,
		Description:        "Sync by applying any detected changes",
		Rollback:           false,
		Metadata:           map[string]string{},
		AvailableOperation: sdk.ManualOperationNone,
	})

	if input.Request.Rollback {
		stages = append(stages, sdk.QuickSyncStage{
			Name:               stageRollback,
			Description:        "Rollback by applying the previous OpenTofu files",
			Rollback:           true,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		})
	}
	return &sdk.BuildQuickSyncStagesResponse{
		Stages: stages,
	}, nil
}
