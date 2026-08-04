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

package provider

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/stretchr/testify/assert"

	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

// TestStorageBlobSASSign covers the deterministic branches of StorageBlobSASSign
// that don't require calling out to Azure Storage: an already SAS-signed URL is
// returned as-is, and malformed/incomplete URLs are rejected before any network
// call would be made.
func TestStorageBlobSASSign(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		blobURL   string
		expected  string
		expectErr bool
	}{
		{
			name:     "already SAS-signed URL is returned without modification",
			blobURL:  "https://myaccount.blob.core.windows.net/mycontainer/myblob.zip?sig=abc123",
			expected: "https://myaccount.blob.core.windows.net/mycontainer/myblob.zip?sig=abc123",
		},
		{
			name:      "invalid URL fails to parse",
			blobURL:   "https://example.com/%zz",
			expectErr: true,
		},
		{
			name:      "URL missing a container name is rejected",
			blobURL:   "https://myaccount.blob.core.windows.net/",
			expectErr: true,
		},
		{
			name:      "URL missing a blob name is rejected",
			blobURL:   "https://myaccount.blob.core.windows.net/mycontainer",
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &sdkClient{}
			got, err := c.StorageBlobSASSign(context.Background(), "my-rg", tc.blobURL)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Empty(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestFunctionSwapSlotX_NoSlotNames covers the guard clause that rejects a swap
// request without slot names before any Azure API call would be made.
func TestFunctionSwapSlotX_NoSlotNames(t *testing.T) {
	t.Parallel()

	c := &sdkClient{}
	err := c.FunctionSwapSlotX(context.Background(), "my-rg", "my-func", "", "")

	assert.EqualError(t, err, "at least one of slot names must be specified")
}

func TestParseARMTemplate(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		armTemplate   config.AzureDeployTemplate
		expectErr     bool
		errorContains string
	}{
		{
			name: "valid template and parameters are parsed, non-object parameter entries are skipped",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "template.json",
				DeploymentParameterFile: "parameters.json",
			},
		},
		{
			name: "missing template file",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "missing.json",
				DeploymentParameterFile: "parameters.json",
			},
			expectErr: true,
		},
		{
			name: "missing parameter file",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "template.json",
				DeploymentParameterFile: "missing.json",
			},
			expectErr: true,
		},
		{
			name: "malformed template json",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "malformed_template.json",
				DeploymentParameterFile: "parameters.json",
			},
			expectErr: true,
		},
		{
			name: "malformed parameters json",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "template.json",
				DeploymentParameterFile: "malformed_parameters.json",
			},
			expectErr: true,
		},
		{
			name: "parameters field is not an object",
			armTemplate: config.AzureDeployTemplate{
				DeploymentTemplateFile:  "template.json",
				DeploymentParameterFile: "parameters_not_object.json",
			},
			expectErr:     true,
			errorContains: "not formated correctly",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &sdkClient{logger: logpersistertest.NewTestLogPersister(t)}
			templateJSON, params, err := c.parseARMTemplate("testdata", tc.armTemplate)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, templateJSON)
				assert.Nil(t, params)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "1.0.0.0", (*templateJSON)["contentVersion"])

			deployParams := *params
			assert.Equal(t, &armresources.DeploymentParameter{Value: "my-func"}, deployParams["functionName"])
			assert.Equal(t, &armresources.DeploymentParameter{Value: float64(3)}, deployParams["instanceCount"])
			assert.NotContains(t, deployParams, "skippedBecauseNotAnObject")
		})
	}
}
