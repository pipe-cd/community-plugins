package deployment

import (
	"context"
	"fmt"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	toolRegistryPkg "github.com/pipe-cd/community-plugins/plugins/sqldef/toolregistry"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func (p *Plugin) executePlanStage(ctx context.Context, dts []*sdk.DeployTarget[config.DeployTargetConfig], input *sdk.ExecuteStageInput[config.ApplicationConfigSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Start planning the schema deployment")

	// Currently, we create them every time the stage is executed beucause we can't pass input.Client.toolRegistry to the plugin when starting the plugin.
	toolRegistry := toolRegistryPkg.NewRegistry(input.Client.ToolRegistry())

	// map for prevent repeatedly download sqldef
	downloadedSqldefPaths := make(map[config.DBType]string)

	for _, dt := range dts {
		lp.Infof("Deploy Target [%s]: host=%s, port=%s, db=%s, schemaFile=%s",
			dt.Name,
			dt.Config.Host,
			dt.Config.Port,
			dt.Config.DBName,
		)

		// check db_type from dt.config to choose which sqldef binary to download
		var sqlDefPath string
		var existed bool
		if sqlDefPath, existed = downloadedSqldefPaths[dt.Config.DbType]; !existed {
			binPath, err := toolRegistry.DownloadBinary(ctx, dt.Config.DbType)
			if err != nil {
				lp.Errorf("Failed while getting Sqldef tool (%v)", err)
				return sdk.StageStatusFailure
			}
			downloadedSqldefPaths[dt.Config.DbType] = binPath
			sqlDefPath = binPath
			lp.Info(fmt.Sprintf("Sqldef binary downloaded: %s", sqlDefPath))
		}

		appDir := input.Request.RunningDeploymentSource.ApplicationDirectory
		schemaPath, err := findFirstSQLFile(appDir)
		if err != nil {
			lp.Errorf("Failed while finding schema file (%v)", err)
			return sdk.StageStatusFailure
		}

		p.Sqldef.Init(lp, dt.Config.Username, dt.Config.Password, dt.Config.Host, dt.Config.Port, dt.Config.DBName, schemaPath, sqlDefPath)

		if err := p.Sqldef.Execute(ctx, true); err != nil {
			lp.Errorf("Failed while plan the deployment (%v)", err)
			return sdk.StageStatusFailure
		}
	}

	return sdk.StageStatusSuccess
}
