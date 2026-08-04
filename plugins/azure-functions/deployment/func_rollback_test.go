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
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

func TestPlugin_executeAzureFuncRollbackStage_InvalidDeployTargetCount(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		dts  []*sdk.DeployTarget[config.AzureDeployTargetConfig]
	}{
		{
			name: "empty deploy targets",
			dts:  []*sdk.DeployTarget[config.AzureDeployTargetConfig]{},
		},
		{
			name: "multiple deploy targets",
			dts: []*sdk.DeployTarget[config.AzureDeployTargetConfig]{
				{Name: "dt-1"},
				{Name: "dt-2"},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := &Plugin{}
			input := newExecuteStageInput(t, string(AzureFuncRollback))

			status := p.executeAzureFuncRollbackStage(context.Background(), tc.dts, input)

			assert.Equal(t, sdk.StageStatusFailure, status)
		})
	}
}
