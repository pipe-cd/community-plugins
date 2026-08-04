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

package livestate

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

func TestGetFunctionLiveState(t *testing.T) {
	t.Parallel()

	manifest := &config.FunctionsSpec{
		FunctionName:      "my-func",
		ResourceGroupName: "my-rg",
		ArmTemplate: &config.AzureDeployTemplate{
			DeploymentName: "my-deployment",
		},
	}

	testcases := []struct {
		name          string
		setupMock     func(m *mockAzureClient)
		expectErr     bool
		expectedState *sdk.ApplicationLiveState
	}{
		{
			name: "function not found reports an unhealthy resource",
			setupMock: func(m *mockAzureClient) {
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(nil, nil)
			},
			expectedState: &sdk.ApplicationLiveState{
				Resources: []sdk.ResourceState{
					{
						Name:              "my-func",
						HealthStatus:      sdk.ResourceHealthStateUnhealthy,
						HealthDescription: "NotFound",
					},
				},
			},
		},
		{
			name: "running function without slots is healthy",
			setupMock: func(m *mockAzureClient) {
				site := &armappservice.Site{
					ID:   to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func"),
					Name: to.Ptr("my-func"),
					Properties: &armappservice.SiteProperties{
						State: to.Ptr("running"),
					},
				}
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(site, nil)
				m.On("FunctionListSlots", mock.Anything, "my-rg", "my-func").Return([]*armappservice.Site{}, nil)
			},
			expectedState: &sdk.ApplicationLiveState{
				Resources: []sdk.ResourceState{
					{
						Name:              "my-func",
						ID:                "/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func",
						ResourceType:      "function",
						HealthStatus:      sdk.ResourceHealthStateHealthy,
						HealthDescription: "",
						DeployTarget:      "my-deployment",
					},
				},
			},
		},
		{
			name: "stopped function is unhealthy with the state as description",
			setupMock: func(m *mockAzureClient) {
				site := &armappservice.Site{
					ID:   to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func"),
					Name: to.Ptr("my-func"),
					Properties: &armappservice.SiteProperties{
						State: to.Ptr("stopped"),
					},
				}
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(site, nil)
				m.On("FunctionListSlots", mock.Anything, "my-rg", "my-func").Return([]*armappservice.Site{}, nil)
			},
			expectedState: &sdk.ApplicationLiveState{
				Resources: []sdk.ResourceState{
					{
						Name:              "my-func",
						ID:                "/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func",
						ResourceType:      "function",
						HealthStatus:      sdk.ResourceHealthStateUnhealthy,
						HealthDescription: "stopped",
						DeployTarget:      "my-deployment",
					},
				},
			},
		},
		{
			name: "function with slots reports each slot as a child resource",
			setupMock: func(m *mockAzureClient) {
				site := &armappservice.Site{
					ID:   to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func"),
					Name: to.Ptr("my-func"),
					Properties: &armappservice.SiteProperties{
						State: to.Ptr("running"),
					},
				}
				slot := &armappservice.Site{
					ID:   to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func/slots/staging"),
					Name: to.Ptr("staging"),
					Properties: &armappservice.SiteProperties{
						State: to.Ptr("stopped"),
					},
				}
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(site, nil)
				m.On("FunctionListSlots", mock.Anything, "my-rg", "my-func").Return([]*armappservice.Site{slot}, nil)
			},
			expectedState: &sdk.ApplicationLiveState{
				Resources: []sdk.ResourceState{
					{
						Name:              "my-func",
						ID:                "/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func",
						ResourceType:      "function",
						HealthStatus:      sdk.ResourceHealthStateHealthy,
						HealthDescription: "",
						DeployTarget:      "my-deployment",
					},
					{
						Name:              "staging",
						ID:                "/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func/slots/staging",
						ParentIDs:         []string{"/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func"},
						ResourceType:      "slot",
						HealthStatus:      sdk.ResourceHealthStateUnhealthy,
						HealthDescription: "stopped",
						DeployTarget:      "my-deployment",
					},
				},
			},
		},
		{
			name: "error from FunctionGetX is propagated",
			setupMock: func(m *mockAzureClient) {
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(nil, errors.New("boom"))
			},
			expectErr: true,
		},
		{
			name: "error from FunctionListSlots is propagated",
			setupMock: func(m *mockAzureClient) {
				site := &armappservice.Site{
					ID:   to.Ptr("id"),
					Name: to.Ptr("my-func"),
					Properties: &armappservice.SiteProperties{
						State: to.Ptr("running"),
					},
				}
				m.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(site, nil)
				m.On("FunctionListSlots", mock.Anything, "my-rg", "my-func").Return(nil, errors.New("boom"))
			},
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &mockAzureClient{}
			tc.setupMock(client)

			got, err := getFunctionLiveState(context.Background(), manifest, client)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				client.AssertExpectations(t)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedState, got)
			client.AssertExpectations(t)
		})
	}
}

func TestGetFunctionSyncState(t *testing.T) {
	t.Parallel()

	manifest := &config.FunctionsSpec{
		ResourceGroupName: "my-rg",
		ArmTemplate: &config.AzureDeployTemplate{
			DeploymentName: "my-deployment",
		},
	}

	testcases := []struct {
		name           string
		manifest       *config.FunctionsSpec
		ds             sdk.DeploymentSource[config.AzureApplicationSpec]
		setupMock      func(m *mockAzureClient)
		expectErr      bool
		expectedStatus sdk.ApplicationSyncStatus
		expectReason   bool
	}{
		{
			name:           "nil manifest is unsupported",
			manifest:       nil,
			expectedStatus: sdk.ApplicationSyncStateUnknown,
		},
		{
			name:           "manifest without an arm template is unsupported",
			manifest:       &config.FunctionsSpec{ResourceGroupName: "my-rg"},
			expectedStatus: sdk.ApplicationSyncStateUnknown,
		},
		{
			name:     "error from DeployARMTemplateWhatIf is propagated",
			manifest: manifest,
			setupMock: func(m *mockAzureClient) {
				m.On("DeployARMTemplateWhatIf", mock.Anything, "my-rg", "", *manifest.ArmTemplate).Return(nil, errors.New("boom"))
			},
			expectErr: true,
		},
		{
			name:     "no changes means the application is synced",
			manifest: manifest,
			setupMock: func(m *mockAzureClient) {
				m.On("DeployARMTemplateWhatIf", mock.Anything, "my-rg", "", *manifest.ArmTemplate).Return(&armresources.WhatIfOperationProperties{}, nil)
			},
			expectedStatus: sdk.ApplicationSyncStateSynced,
		},
		{
			name:     "create, delete and modify changes report out of sync with a truncated commit hash",
			manifest: manifest,
			ds: sdk.DeploymentSource[config.AzureApplicationSpec]{
				CommitHash: "0123456789abcdef",
			},
			setupMock: func(m *mockAzureClient) {
				props := &armresources.WhatIfOperationProperties{
					Changes: []*armresources.WhatIfChange{
						{
							ChangeType: to.Ptr(armresources.ChangeTypeCreate),
							ResourceID: to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/new-func"),
						},
						{
							ChangeType: to.Ptr(armresources.ChangeTypeDelete),
							ResourceID: to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/old-func"),
						},
						{
							// Modify with no delta entries: exercises the counter/branch
							// without recursing into per-property rendering.
							ChangeType: to.Ptr(armresources.ChangeTypeModify),
							ResourceID: to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-func"),
							Delta:      []*armresources.WhatIfPropertyChange{},
						},
					},
				}
				m.On("DeployARMTemplateWhatIf", mock.Anything, "my-rg", "", *manifest.ArmTemplate).Return(props, nil)
			},
			expectedStatus: sdk.ApplicationSyncStateOutOfSync,
			expectReason:   true,
		},
		{
			name:     "potential changes alone still count as out of sync",
			manifest: manifest,
			ds: sdk.DeploymentSource[config.AzureApplicationSpec]{
				CommitHash: "short",
			},
			setupMock: func(m *mockAzureClient) {
				props := &armresources.WhatIfOperationProperties{
					PotentialChanges: []*armresources.WhatIfChange{
						{
							ChangeType: to.Ptr(armresources.ChangeTypeCreate),
							ResourceID: to.Ptr("/subscriptions/x/resourceGroups/my-rg/providers/Microsoft.Web/sites/new-func"),
						},
					},
				}
				m.On("DeployARMTemplateWhatIf", mock.Anything, "my-rg", "", *manifest.ArmTemplate).Return(props, nil)
			},
			expectedStatus: sdk.ApplicationSyncStateOutOfSync,
			expectReason:   true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &mockAzureClient{}
			if tc.setupMock != nil {
				tc.setupMock(client)
			}

			got, err := getFunctionSyncState(context.Background(), tc.ds, tc.manifest, client)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				client.AssertExpectations(t)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, got.Status)
			if tc.expectReason {
				assert.NotEmpty(t, got.Reason)
				assert.NotEmpty(t, got.ShortReason)
			}
			client.AssertExpectations(t)
		})
	}
}

func TestGetFunction(t *testing.T) {
	t.Parallel()

	manifest := &config.FunctionsSpec{
		FunctionName:      "my-func",
		ResourceGroupName: "my-rg",
	}

	t.Run("success populates both live state and sync state", func(t *testing.T) {
		t.Parallel()

		client := &mockAzureClient{}
		client.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(nil, nil)

		input := &sdk.GetLivestateInput[config.AzureApplicationSpec]{Logger: zap.NewNop()}

		resp, err := getFunction(context.Background(), manifest, client, input)

		assert.NoError(t, err)
		assert.Len(t, resp.LiveState.Resources, 1)
		assert.Equal(t, sdk.ApplicationSyncStateUnknown, resp.SyncState.Status)
		client.AssertExpectations(t)
	})

	t.Run("live state error is logged and does not fail the call", func(t *testing.T) {
		t.Parallel()

		client := &mockAzureClient{}
		client.On("FunctionGetX", mock.Anything, "my-rg", "my-func", "").Return(nil, errors.New("boom"))

		input := &sdk.GetLivestateInput[config.AzureApplicationSpec]{Logger: zap.NewNop()}

		resp, err := getFunction(context.Background(), manifest, client, input)

		assert.NoError(t, err)
		assert.Empty(t, resp.LiveState.Resources)
		client.AssertExpectations(t)
	})
}

func TestPlugin_GetLivestate_InvalidDeployTargetCount(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		dts  []*sdk.DeployTarget[config.AzureDeployTargetConfig]
	}{
		{name: "no deploy targets", dts: []*sdk.DeployTarget[config.AzureDeployTargetConfig]{}},
		{name: "multiple deploy targets", dts: []*sdk.DeployTarget[config.AzureDeployTargetConfig]{{Name: "a"}, {Name: "b"}}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := Plugin{}
			got, err := p.GetLivestate(context.Background(), nil, tc.dts, &sdk.GetLivestateInput[config.AzureApplicationSpec]{})

			assert.Error(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestPlugin_GetLivestate_MissingApplicationConfig(t *testing.T) {
	t.Parallel()

	p := Plugin{}
	dts := []*sdk.DeployTarget[config.AzureDeployTargetConfig]{{Name: "dt"}}
	input := &sdk.GetLivestateInput[config.AzureApplicationSpec]{
		Request: sdk.GetLivestateRequest[config.AzureApplicationSpec]{
			DeploymentSource: sdk.DeploymentSource[config.AzureApplicationSpec]{},
		},
	}

	got, err := p.GetLivestate(context.Background(), nil, dts, input)

	assert.Error(t, err)
	assert.Nil(t, got)
}
