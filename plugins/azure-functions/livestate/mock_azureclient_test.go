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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/stretchr/testify/mock"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

// mockAzureClient is a mock implementation of provider.AzureClient, used to
// exercise the livestate business logic without reaching out to Azure.
type mockAzureClient struct {
	mock.Mock
}

func (m *mockAzureClient) FunctionGetX(ctx context.Context, resourceGroupName, functionName, slotName string) (*armappservice.Site, error) {
	args := m.Called(ctx, resourceGroupName, functionName, slotName)
	site, _ := args.Get(0).(*armappservice.Site)
	return site, args.Error(1)
}

func (m *mockAzureClient) FunctionListSlots(ctx context.Context, resourceGroupName, functionName string) ([]*armappservice.Site, error) {
	args := m.Called(ctx, resourceGroupName, functionName)
	slots, _ := args.Get(0).([]*armappservice.Site)
	return slots, args.Error(1)
}

func (m *mockAzureClient) FunctionSwapSlotX(ctx context.Context, resourceGroupName, functionName, slotName1, slotName2 string) error {
	args := m.Called(ctx, resourceGroupName, functionName, slotName1, slotName2)
	return args.Error(0)
}

func (m *mockAzureClient) FunctionValidate(ctx context.Context, resourceGroupName, functionName string, slotNames []string) (bool, error) {
	args := m.Called(ctx, resourceGroupName, functionName, slotNames)
	return args.Bool(0), args.Error(1)
}

func (m *mockAzureClient) FunctionKuduDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, blobURL string) error {
	args := m.Called(ctx, resourceGroupName, functionName, slotName, blobURL)
	return args.Error(0)
}

func (m *mockAzureClient) FunctionRunFromPackageDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, blobURL string) error {
	args := m.Called(ctx, resourceGroupName, functionName, slotName, blobURL)
	return args.Error(0)
}

func (m *mockAzureClient) DeployARMTemplate(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) error {
	args := m.Called(ctx, resourceGroupName, appDir, armTemplate)
	return args.Error(0)
}

func (m *mockAzureClient) DeployARMTemplateWhatIf(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) (*armresources.WhatIfOperationProperties, error) {
	args := m.Called(ctx, resourceGroupName, appDir, armTemplate)
	props, _ := args.Get(0).(*armresources.WhatIfOperationProperties)
	return props, args.Error(1)
}
