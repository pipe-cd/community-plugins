package deployment

import (
	"context"
	"fmt"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/provider"
	toolRegistryPkg "github.com/pipe-cd/community-plugins/plugins/sqldef/toolregistry"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func (p *Plugin) executeSqldefSyncStage(ctx context.Context, dts []*sdk.DeployTarget[config.SqldefDeployTargetConfig], input *sdk.ExecuteStageInput[config.SqldefApplicationSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()
	lp.Info("Start syncing the deployment")
	// Currently, we create them every time the stage is executed beucause we can't pass input.Client.toolRegistry to the plugin when starting the plugin.
	toolRegistry := toolRegistryPkg.NewRegistry(input.Client.ToolRegistry())

	for _, dt := range dts {
		// TODO: check db_type from dt.config to choose which sqldef binary to download
		// Now we temporarily hard-coded as mysql
		sqlDefPath, err := toolRegistry.Mysqldef(ctx, "")

		lp.Info(fmt.Sprintf("Sqldef binary downloaded: %s", sqlDefPath))
		if err != nil {
			lp.Errorf("Failed while getting Sqldef tool (%v)", err)
			return sdk.StageStatusFailure
		}

		lp.Info(fmt.Sprintf("dt: %+v\n", dt))

		appDir := input.Request.RunningDeploymentSource.ApplicationDirectory
		lp.Info(fmt.Sprintf("appDir: %s", appDir))

		schemaPath := fmt.Sprintf("%s/%s", appDir, dt.Config.SchemaFileName)

		sqldef := provider.NewSqldef(lp, dt.Config.Username, dt.Config.Password, dt.Config.Host, dt.Config.Port, dt.Config.DBName, schemaPath, sqlDefPath)

		err = sqldef.Execute(ctx, dt.Config.DryRun, dt.Config.EnableDrop)
		if err != nil {
			lp.Errorf("Failed while syncing the deployment (%v)", err)
		}
	}

	return sdk.StageStatusSuccess
}
