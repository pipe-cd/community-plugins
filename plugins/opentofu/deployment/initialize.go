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

	"github.com/pipe-cd/community-plugins/plugins/opentofu/config"
	"github.com/pipe-cd/community-plugins/plugins/opentofu/provider"
	"github.com/pipe-cd/community-plugins/plugins/opentofu/toolregistry"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func initOpenTofuCommand(ctx context.Context, client *sdk.Client, ds sdk.DeploymentSource[config.ApplicationConfigSpec], dt *sdk.DeployTarget[config.DeployTargetConfig]) (*provider.OpenTofu, error) {
	var (
		appSpec = ds.ApplicationConfig.Spec
		flags   = appSpec.CommandFlags
		envs    = appSpec.CommandEnvs
		lp      = client.LogPersister()
	)
	tr := toolregistry.NewRegistry(client.ToolRegistry())
	opentofuPath, err := tr.OpenTofu(ctx, appSpec.OpenTofuVersion)
	if err != nil {
		lp.Errorf("Failed to find opentofu (%v)", err)
		return nil, err
	}

	cmd := provider.NewOpenTofu(
		opentofuPath,
		ds.ApplicationDirectory,
		provider.WithVars(mergeVars(dt.Config.Vars, appSpec.Vars)),
		provider.WithVarFiles(appSpec.VarFiles),
		provider.WithAdditionalFlags(flags.Shared, flags.Init, flags.Plan, flags.Apply),
		provider.WithAdditionalEnvs(envs.Shared, envs.Init, envs.Plan, envs.Apply),
	)

	if ok := showUsingVersion(ctx, cmd, lp); !ok {
		return nil, errors.New("failed to show using version")
	}

	if err := cmd.Init(ctx, lp); err != nil {
		lp.Errorf("Failed to execute 'tofu init' (%v)", err)
		return nil, err
	}

	if ok := selectWorkspace(ctx, cmd, appSpec.Workspace, lp); !ok {
		return nil, errors.New("failed to select workspace")
	}

	return cmd, nil
}

func mergeVars(deployTargetVars []string, appVars []string) []string {
	// TODO: Validate duplication
	mergedVars := make([]string, 0, len(deployTargetVars)+len(appVars))
	mergedVars = append(mergedVars, deployTargetVars...)
	mergedVars = append(mergedVars, appVars...)
	return mergedVars
}

func showUsingVersion(ctx context.Context, cmd *provider.OpenTofu, lp sdk.StageLogPersister) bool {
	version, err := cmd.Version(ctx)
	if err != nil {
		lp.Errorf("Failed to check opentofu version (%v)", err)
		return false
	}
	lp.Infof("Using opentofu version %q to execute the opentofu commands", version)
	return true
}

func selectWorkspace(ctx context.Context, cmd *provider.OpenTofu, workspace string, lp sdk.StageLogPersister) bool {
	if workspace == "" {
		return true
	}
	if err := cmd.SelectWorkspace(ctx, workspace); err != nil {
		lp.Errorf("Failed to select workspace %q (%v). You might need to create the workspace before using by command %q", workspace, err, "opentofu workspace new "+workspace)
		return false
	}
	lp.Infof("Selected workspace %q", workspace)
	return true
}
