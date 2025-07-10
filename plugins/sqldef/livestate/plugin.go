package livestate

import (
	"context"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

type Plugin struct{}

// GetLivestate implements sdk.LivestatePlugin.
func (p Plugin) GetLivestate(ctx context.Context, _ *sdk.ConfigNone, deployTargets []*sdk.DeployTarget[config.SqldefDeployTargetConfig], input *sdk.GetLivestateInput[config.SqldefApplicationSpec]) (*sdk.GetLivestateResponse, error) {
	// MOCK first, TODO
	return &sdk.GetLivestateResponse{
		LiveState: sdk.ApplicationLiveState{
			Resources: []sdk.ResourceState{},
		},
		SyncState: sdk.ApplicationSyncState{
			Status: sdk.ApplicationSyncStateUnknown,
		},
	}, nil
}
