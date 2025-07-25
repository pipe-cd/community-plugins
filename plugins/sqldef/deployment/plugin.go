package deployment

import (
	"context"
	"slices"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
)

const (
	// run sqldef with --dry-run
	sqldefStagePlan  string = "SQLDEF_PLAN"
	sqldefStageApply string = "SQLDEF_APPLY"
	// by running with previous DB schema dump
	sqldefStageRollback string = "SQLDEF_ROLLBACK"
)

// Plugin implements sdk.DeploymentPlugin for OpenTofu.
type Plugin struct{}

var _ sdk.DeploymentPlugin[config.Config, config.DeployTargetConfig, config.ApplicationConfigSpec] = (*Plugin)(nil)

func (*Plugin) FetchDefinedStages() []string {
	return []string{
		sqldefStagePlan,
		sqldefStageApply,
		sqldefStageRollback,
	}
}

// BuildPipelineSyncStages builds the stages that will be executed by the plugin.
func (p *Plugin) BuildPipelineSyncStages(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.BuildPipelineSyncStagesInput,
) (*sdk.BuildPipelineSyncStagesResponse, error) {
	reqStages := input.Request.Stages
	out := make([]sdk.PipelineStage, 0, len(reqStages)+1)

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
		// we set the index of the rollback stage to the minimum index of all stages.
		minIndex := slices.MinFunc(reqStages, func(a, b sdk.StageConfig) int { return a.Index - b.Index }).Index
		out = append(out, sdk.PipelineStage{
			Name:               sqldefStageRollback,
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
	// show the commit hash as the version
	return &sdk.DetermineVersionsResponse{
		Versions: []sdk.ArtifactVersion{
			{Version: input.Request.DeploymentSource.CommitHash},
		},
	}, nil
}

// No need for the sqldef plugin
// DetermineStrategy determines the strategy for the deployment.
func (p *Plugin) DetermineStrategy(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.DetermineStrategyInput[config.ApplicationConfigSpec],
) (*sdk.DetermineStrategyResponse, error) {
	return nil, nil
}

// BuildQuickSyncStages builds the stages for quick sync.
func (p *Plugin) BuildQuickSyncStages(
	ctx context.Context,
	cfg *config.Config,
	input *sdk.BuildQuickSyncStagesInput,
) (*sdk.BuildQuickSyncStagesResponse, error) {
	stages := make([]sdk.QuickSyncStage, 0, 2)
	stages = append(stages, sdk.QuickSyncStage{
		Name:               sqldefStageApply,
		Description:        "Apply changes to target DB",
		Rollback:           false,
		Metadata:           map[string]string{},
		AvailableOperation: sdk.ManualOperationNone,
	})

	if input.Request.Rollback {
		stages = append(stages, sdk.QuickSyncStage{
			Name:               sqldefStageRollback,
			Description:        "Rollback to previous DB schema",
			Rollback:           true,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		})
	}

	return &sdk.BuildQuickSyncStagesResponse{
		Stages: stages,
	}, nil
}
