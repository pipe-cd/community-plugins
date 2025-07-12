package deployment

import (
	"context"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
)

// Plugin implements sdk.DeploymentPlugin for OpenTofu.
type Plugin struct{}

var _ sdk.DeploymentPlugin[config.Config, config.DeployTargetConfig, config.ApplicationConfigSpec] = (*Plugin)(nil)

func (*Plugin) FetchDefinedStages() []string {
	// TODO: Implement FetchDefinedStages logic
	return []string{}
}

// BuildPipelineSyncStages builds the stages that will be executed by the plugin.
func (p *Plugin) BuildPipelineSyncStages(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.BuildPipelineSyncStagesInput,
) (*sdk.BuildPipelineSyncStagesResponse, error) {
	// TODO: Implement BuildPipelineSyncStages logic
	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: []sdk.PipelineStage{},
	}, nil
}

// ExecuteStage executes the given stage.
func (p *Plugin) ExecuteStage(
	ctx context.Context,
	cfg *config.Config,
	dts []*sdk.DeployTarget[config.DeployTargetConfig],
	input *sdk.ExecuteStageInput[config.ApplicationConfigSpec],
) (*sdk.ExecuteStageResponse, error) {
	// TODO: Implement ExecuteStage logic
	return &sdk.ExecuteStageResponse{
		Status: sdk.StageStatusSuccess,
	}, nil
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
func (p *Plugin) BuildQuickSyncStages(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.BuildQuickSyncStagesInput,
) (*sdk.BuildQuickSyncStagesResponse, error) {
	// TODO: Implement BuildQuickSyncStages logic
	return &sdk.BuildQuickSyncStagesResponse{
		Stages: []sdk.QuickSyncStage{},
	}, nil
}
