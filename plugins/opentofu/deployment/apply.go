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
	"encoding/json"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/opentofu/config"
)

func (p *Plugin) executeApplyStage(ctx context.Context, input *sdk.ExecuteStageInput[config.ApplicationConfigSpec], dts []*sdk.DeployTarget[config.DeployTargetConfig]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Starting OpenTofu apply stage")

	cmd, err := initOpenTofuCommand(ctx, input.Client, input.Request.TargetDeploymentSource, dts[0])
	if err != nil {
		lp.Errorf("Failed to initialize OpenTofu command: %v", err)
		return sdk.StageStatusFailure
	}

	var stageConfig config.OpenTofuApplyStageOptions
	if err := json.Unmarshal(input.Request.StageConfig, &stageConfig); err != nil {
		lp.Errorf("Failed to unmarshal stage config (%v)", err)
		return sdk.StageStatusFailure
	}

	lp.Infof("Starting Apply stage. AutoApprove: %v", stageConfig.AutoApprove)

	if err := cmd.Apply(ctx, lp); err != nil {
		lp.Errorf("Failed to Apply (%v)", err)
		return sdk.StageStatusFailure
	}

	lp.Success("Successfully applied changes")
	return sdk.StageStatusSuccess
}
