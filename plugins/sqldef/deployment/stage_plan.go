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
		// log deploy target info
		lp.Infof("Deploy Target [%s]: host=%s, port=%s, db=%s, schemaFile=%s",
			dt.Name,
			dt.Config.Host,
			dt.Config.Port,
			dt.Config.DBName,
			dt.Config.SchemaFileName,
		)

		// check dt.Config.SchemaFileName
		if dt.Config.SchemaFileName == "" {
			lp.Errorf("Configuration error: SchemaFileName is empty. Please ensure 'SchemaFileName' is properly set in dt.Config before running this stage.")
			return sdk.StageStatusFailure
		}

		// check db_type from dt.config to choose which sqldef binary to download
		// Now we only support mysql temporarily
		var sqlDefPath string
		var existed bool
		if sqlDefPath, existed = downloadedSqldefPaths[config.DBTypeMySQL]; !existed {
			var binPath string
			var err error
			switch dt.Config.DbType {
			case config.DBTypeMySQL:
				binPath, err = toolRegistry.Mysqldef(ctx, "")
			default:
				lp.Errorf(fmt.Sprintf("Unsupported database type: %s, currently only support: mysql", dt.Name))
				return sdk.StageStatusFailure
			}
			if err != nil {
				lp.Errorf("Failed while getting Sqldef tool (%v)", err)
				return sdk.StageStatusFailure
			}
			downloadedSqldefPaths[dt.Config.DbType] = binPath
			lp.Info(fmt.Sprintf("Sqldef binary downloadeded: %s", sqlDefPath))
		}

		appDir := input.Request.RunningDeploymentSource.ApplicationDirectory
		schemaPath := fmt.Sprintf("%s/%s", appDir, dt.Config.SchemaFileName)
		p.sqldef.Init(lp, dt.Config.Username, dt.Config.Password, dt.Config.Host, dt.Config.Port, dt.Config.DBName, schemaPath, sqlDefPath)

		if err := p.sqldef.Execute(ctx, true); err != nil {
			lp.Errorf("Failed while plan the deployment (%v)", err)
		}
	}

	return sdk.StageStatusSuccess
}
