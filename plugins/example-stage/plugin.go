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

package main

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

type plugin struct{}

// ensure the type plugin implements sdk.StagePlugin.
var _ sdk.StagePlugin[pluginConfig, sdk.ConfigNone, sdk.ConfigNone] = (*plugin)(nil)

// pluginConfig is the config for the plugin.
type pluginConfig struct {
	CommonMessage string `json:"commonMessage"`
}

// helloStageOptions is the options for the EXAMPLE_HELLO stage.
type helloStageOptions struct {
	Name string `json:"name"`
}

// goodbyeStageOptions is the options for the EXAMPLE_GOODBYE stage.
type goodbyeStageOptions struct {
	Message string `json:"message"`
}

func (p *helloStageOptions) validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

const (
	stageExampleHello   = "EXAMPLE_HELLO"
	stageExampleGoodbye = "EXAMPLE_GOODBYE"
)

// FetchDefinedStages returns the list of stages that the plugin can execute. This implements sdk.StagePlugin.
func (p *plugin) FetchDefinedStages() []string {
	return []string{
		stageExampleHello,
		stageExampleGoodbye,
	}
}

// BuildPipelineSyncStages builds the stages that will be executed by the plugin. This implements sdk.StagePlugin.
func (p *plugin) BuildPipelineSyncStages(ctx context.Context, _ *pluginConfig, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: buildPipelineSyncStages(input.Request),
	}, nil
}

// ExecuteStage executes the given stage. This implements sdk.StagePlugin.
func (p *plugin) ExecuteStage(ctx context.Context, cfg *pluginConfig, _ []*sdk.DeployTarget[sdk.ConfigNone], input *sdk.ExecuteStageInput[sdk.ConfigNone]) (*sdk.ExecuteStageResponse, error) {
	switch input.Request.StageName {
	case stageExampleHello:
		return executeHello(cfg, input)
	case stageExampleGoodbye:
		return executeGoodbye(cfg, input)
	}
	return nil, fmt.Errorf("unsupported stage: %s", input.Request.StageName)
}

// buildPipelineSyncStages builds the stages that will be executed by the plugin.
func buildPipelineSyncStages(req sdk.BuildPipelineSyncStagesRequest) []sdk.PipelineStage {
	stages := make([]sdk.PipelineStage, 0, len(req.Stages))
	for _, rs := range req.Stages {
		stage := sdk.PipelineStage{
			Index:              rs.Index,
			Name:               rs.Name,
			Rollback:           false,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		}
		stages = append(stages, stage)
	}
	return stages
}

// executeHello executes the HELLO stage. It will send a message to the UI.
func executeHello(cfg *pluginConfig, input *sdk.ExecuteStageInput[sdk.ConfigNone]) (*sdk.ExecuteStageResponse, error) {
	lp := input.Client.LogPersister()
	var stageOpts helloStageOptions
	if err := json.Unmarshal(input.Request.StageConfig, &stageOpts); err != nil {
		lp.Errorf("Failed to unmarshal the stage config (%v)", err)
		return nil, fmt.Errorf("failed to unmarshal the stage config (%v)", err)
	}
	if err := stageOpts.validate(); err != nil {
		lp.Errorf("Invalid stage options: %v", err)
		return nil, fmt.Errorf("invalid stage options: %v", err)
	}

	lp.Infof("Hello %s from the example HELLO stage!\n CommonMessage: %s", stageOpts.Name, cfg.CommonMessage)
	return &sdk.ExecuteStageResponse{Status: sdk.StageStatusSuccess}, nil
}

// executeGoodbye executes the GOODBYE stage. It will send a message to the UI.
func executeGoodbye(cfg *pluginConfig, input *sdk.ExecuteStageInput[sdk.ConfigNone]) (*sdk.ExecuteStageResponse, error) {
	lp := input.Client.LogPersister()
	var stageOpts goodbyeStageOptions
	if err := json.Unmarshal(input.Request.StageConfig, &stageOpts); err != nil {
		lp.Errorf("Failed to unmarshal the stage config (%v)", err)
		return nil, fmt.Errorf("failed to unmarshal the stage config (%v)", err)
	}

	lp.Infof("Goodbye from the example GOODBYE stage!\n Message: %s\n CommonMessage: %s", stageOpts.Message, cfg.CommonMessage)
	return &sdk.ExecuteStageResponse{Status: sdk.StageStatusSuccess}, nil
}
