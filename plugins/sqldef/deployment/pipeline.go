package deployment

import (
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

const (
	SqldefSync     string = "SQLDEF_SYNC"
	SqldefRollback string = "SQLDEF_ROLLBACK"
)

var allStages = []string{
	SqldefSync,
	SqldefRollback,
}

// buildPipelineStages builds the pipeline stages with the given SDK stages.
func buildPipelineStages(stages []sdk.StageConfig, autoRollback bool) []sdk.PipelineStage {
	// MOCK first, TODO
	out := make([]sdk.PipelineStage, 0, len(stages)+1)

	for _, s := range stages {
		out = append(out, sdk.PipelineStage{
			Name:               s.Name,
			Index:              s.Index,
			Rollback:           false,
			Metadata:           make(map[string]string, 0),
			AvailableOperation: sdk.ManualOperationNone,
		})
	}

	//if autoRollback {
	//	// we set the index of the rollback stage to the minimum index of all stages.
	//	minIndex := slices.MinFunc(stages, func(a, b sdk.StageConfig) int {
	//		return a.Index - b.Index
	//	}).Index
	//
	//	out = append(out, sdk.PipelineStage{
	//		Name:               SqldefRollback,
	//		Index:              minIndex,
	//		Rollback:           true,
	//		Metadata:           make(map[string]string, 0),
	//		AvailableOperation: sdk.ManualOperationNone,
	//	})
	//}

	return out
}

func buildQuickSync(autoRollback bool) []sdk.QuickSyncStage {
	// MOCK first, TODO
	out := make([]sdk.QuickSyncStage, 0, 2)
	out = append(out, sdk.QuickSyncStage{
		Name:               SqldefSync,
		Description:        "", //TODO: add description
		Metadata:           map[string]string{},
		AvailableOperation: sdk.ManualOperationNone,
	})
	if autoRollback {
		out = append(out, sdk.QuickSyncStage{
			Name:               string(SqldefRollback),
			Description:        "", // TODO
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
			Rollback:           true,
		})
	}
	return out
}
