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
	"slices"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

type stage string

const (
	AzureFuncSync     stage = "AZURE_FUNCTION_SYNC"
	AzureFuncSwap     stage = "AZURE_FUNCTION_SWAP"
	AzureFuncRollback stage = "AZURE_FUNCTION_ROLLBACK"
)

var allStages = []string{
	string(AzureFuncSync),
	string(AzureFuncSwap),
	string(AzureFuncRollback),
}

func buildQuickSync(autoRollback bool) []sdk.QuickSyncStage {
	out := make([]sdk.QuickSyncStage, 0, 2)
	out = append(out, sdk.QuickSyncStage{
		Name:               string(AzureFuncSync),
		Description:        "", //TODO: add description
		Metadata:           map[string]string{},
		AvailableOperation: sdk.ManualOperationNone,
	})
	if autoRollback {
		out = append(out, sdk.QuickSyncStage{
			Name:               string(AzureFuncRollback),
			Description:        "", // TODO
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
			Rollback:           true,
		})
	}
	return out
}

func buildPipeline(stages []sdk.StageConfig, autoRollback bool) []sdk.PipelineStage {
	out := make([]sdk.PipelineStage, 0, len(stages)+1)
	for _, s := range stages {
		out = append(out, sdk.PipelineStage{
			Name:               s.Name,
			Index:              s.Index,
			Rollback:           false,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		})
	}

	if autoRollback {
		// we set the index of the rollback stage to the minimum index of all stages.
		minIndex := slices.MinFunc(stages, func(a, b sdk.StageConfig) int {
			return a.Index - b.Index
		}).Index

		out = append(out, sdk.PipelineStage{
			Name:               string(AzureFuncRollback),
			Index:              minIndex,
			Rollback:           true,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		})
	}
	return out
}
