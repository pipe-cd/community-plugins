package deployment

import (
	"context"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

type Plugin struct{}

type toolRegistry interface {
	Mysqldef(ctx context.Context, version string) (string, error)
}

// FetchDefinedStages returns the defined stages for this plugin.
func (p *Plugin) FetchDefinedStages() []string {
	return allStages
}

// BuildPipelineSyncStages returns the stages for the pipeline sync strategy.
func (p *Plugin) BuildPipelineSyncStages(ctx context.Context, _ *sdk.ConfigNone, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	// MOCK first, TODO
	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: buildPipelineStages(input.Request.Stages, input.Request.Rollback),
	}, nil
}

// ExecuteStage executes the stage.
func (p *Plugin) ExecuteStage(ctx context.Context, _ *sdk.ConfigNone, dts []*sdk.DeployTarget[config.SqldefDeployTargetConfig], input *sdk.ExecuteStageInput[config.SqldefApplicationSpec]) (*sdk.ExecuteStageResponse, error) {

	switch input.Request.StageName {
	case SqldefSync:
		return &sdk.ExecuteStageResponse{
			Status: p.executeSqldefSyncStage(ctx, dts, input),
		}, nil
	default:
		panic("unimplemented stage: " + input.Request.StageName)
	}
}

// DetermineVersions determines the versions of the application.
func (p *Plugin) DetermineVersions(ctx context.Context, _ *sdk.ConfigNone, input *sdk.DetermineVersionsInput[config.SqldefApplicationSpec]) (*sdk.DetermineVersionsResponse, error) {
	// MOCK first, TODO
	return &sdk.DetermineVersionsResponse{
		Versions: []sdk.ArtifactVersion{
			{
				Name:    "DetermineVersionsResponse",
				Version: "DetermineVersionsResponse",
				URL:     "DetermineVersionsResponse",
			},
		},
	}, nil
}

// DetermineStrategy determines the strategy for the deployment.
func (p *Plugin) DetermineStrategy(ctx context.Context, _ *sdk.ConfigNone, input *sdk.DetermineStrategyInput[config.SqldefApplicationSpec]) (*sdk.DetermineStrategyResponse, error) {
	return nil, nil
}

// BuildQuickSyncStages returns the stages for the quick sync strategy.
func (p *Plugin) BuildQuickSyncStages(ctx context.Context, _ *sdk.ConfigNone, input *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	// MOCK first, TODO
	return &sdk.BuildQuickSyncStagesResponse{
		Stages: buildQuickSync(input.Request.Rollback),
	}, nil
}
