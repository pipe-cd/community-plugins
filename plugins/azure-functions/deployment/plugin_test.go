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
	"testing"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

// Test_ExecuteStage_UnknownStage ensures that an unrecognized stage name
// results in an error instead of a panic, since the stage name originates
// from the pipeline configuration and can be malformed or unsupported.
func Test_ExecuteStage_UnknownStage(t *testing.T) {
	t.Parallel()

	p := &Plugin{}

	got, err := p.ExecuteStage(t.Context(), nil, nil, &sdk.ExecuteStageInput[config.AzureApplicationSpec]{
		Request: sdk.ExecuteStageRequest[config.AzureApplicationSpec]{
			StageName: "UNKNOWN_STAGE",
		},
	})

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "UNKNOWN_STAGE")
}

// Test_DetermineVersions_UnsupportedKind ensures that an application spec
// with an unsupported "kind" results in an error instead of a panic, since
// the kind comes directly from the user-authored application config file.
func Test_DetermineVersions_UnsupportedKind(t *testing.T) {
	t.Parallel()

	p := &Plugin{}

	appCfg := &sdk.ApplicationConfig[config.AzureApplicationSpec]{
		Spec: &config.AzureApplicationSpec{
			Kind: "unsupported-kind",
		},
	}

	got, err := p.DetermineVersions(t.Context(), nil, &sdk.DetermineVersionsInput[config.AzureApplicationSpec]{
		Request: sdk.DetermineVersionsRequest[config.AzureApplicationSpec]{
			DeploymentSource: sdk.DeploymentSource[config.AzureApplicationSpec]{
				ApplicationConfig: appCfg,
			},
		},
	})

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "unsupported-kind")
}

// Test_DetermineVersions_FunctionKind ensures the existing, supported
// behavior for the "function" kind is preserved.
func Test_DetermineVersions_FunctionKind(t *testing.T) {
	t.Parallel()

	p := &Plugin{}

	appCfg := &sdk.ApplicationConfig[config.AzureApplicationSpec]{
		Spec: &config.AzureApplicationSpec{
			Kind: config.FunctionKind,
			FunctionManifest: &config.FunctionsSpec{
				PackageUri: "https://example.blob.core.windows.net/pkg.zip",
			},
		},
	}

	got, err := p.DetermineVersions(t.Context(), nil, &sdk.DetermineVersionsInput[config.AzureApplicationSpec]{
		Request: sdk.DetermineVersionsRequest[config.AzureApplicationSpec]{
			DeploymentSource: sdk.DeploymentSource[config.AzureApplicationSpec]{
				ApplicationConfig: appCfg,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Versions, 1)
	assert.Equal(t, "https://example.blob.core.windows.net/pkg.zip", got.Versions[0].Version)
}
