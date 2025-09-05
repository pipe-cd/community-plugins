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
	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	toolRegistryPkg "github.com/pipe-cd/community-plugins/plugins/sqldef/toolregistry"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func (p *Plugin) executeRollbackStage(ctx context.Context, dts []*sdk.DeployTarget[config.DeployTargetConfig], input *sdk.ExecuteStageInput[config.ApplicationConfigSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Start rollback the schema deployment")

	// Currently, we create them every time the stage is executed because we can't pass input.Client.toolRegistry to the plugin when starting the plugin.
	toolRegistry := toolRegistryPkg.NewRegistry(input.Client.ToolRegistry())

	for _, dt := range dts {
		lp.Infof("Deploy Target [%s]: host=%s, port=%s, db=%s",
			dt.Name,
			dt.Config.Host,
			dt.Config.Port,
			dt.Config.DBName,
		)

		sqlDefPath, err := toolRegistry.DownloadBinary(ctx, dt.Config.DbType, "")
		if err != nil {
			lp.Errorf("Failed while getting Sqldef tool (%v)", err)
			return sdk.StageStatusFailure
		}

		appDir := input.Request.RunningDeploymentSource.ApplicationDirectory
		schemaPath, err := findFirstSQLFile(appDir)
		if err != nil {
			lp.Errorf("Failed while finding schema file (%v)", err)
			return sdk.StageStatusFailure
		}

		p.Sqldef.Init(lp, dt.Config.Username, dt.Config.Password, dt.Config.Host, dt.Config.Port, dt.Config.DBName, schemaPath, sqlDefPath)

		if err := p.Sqldef.Execute(ctx, false); err != nil {
			lp.Errorf("Failed while applying the deployment (%v)", err)
			return sdk.StageStatusFailure
		}
	}

	return sdk.StageStatusSuccess
}
