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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"

	"github.com/pipe-cd/community-plugins/plugins/azure-functions/config"
)

type AzureClient interface {
	FunctionGetX(ctx context.Context, resourceGroupName, functionName, slotName string) (*armappservice.Site, error)
	FunctionListSlots(ctx context.Context, resourceGroupName, functionName string) ([]*armappservice.Site, error)
	FunctionSwapSlotX(ctx context.Context, resourceGroupName, functionName, slotName1, slotName2 string) error
	FunctionValidate(ctx context.Context, resourceGroupName, functionName string, slotNames []string) (bool, error)
	// FunctionKuduDeploymentX https://learn.microsoft.com/en-us/azure/azure-functions/functions-deployment-technologies, https://github.com/projectkudu/kudu/wiki/REST-API
	FunctionKuduDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, blobURL string) error
	// FunctionRunFromPackageDeploymentX  https://learn.microsoft.com/en-us/azure/azure-functions/run-functions-from-deployment-package
	FunctionRunFromPackageDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, blobURL string) error

	DeployARMTemplate(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) error
	DeployARMTemplateWhatIf(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) (*armresources.WhatIfOperationProperties, error)
}

type sdkClient struct {
	cred                 azcore.TokenCredential
	resourceGroupClient  *armresources.ResourceGroupsClient
	accountStorageClient *armstorage.AccountsClient
	deploymentClient     *armresources.DeploymentsClient

	planClient   *armappservice.PlansClient
	webAppClient *armappservice.WebAppsClient // functions, web apps for container

	managedEnvironmentClient *armappcontainers.ManagedEnvironmentsClient // container app
	managedClusterClient     *armcontainerservice.ManagedClustersClient  // aks
	containerGroupClient     *armcontainerinstance.ContainerGroupsClient // aci
	containerAppClient       *armappcontainers.ContainerAppsClient
	logger                   sdk.StageLogPersister
	tags                     map[string]*string
}

func (c *sdkClient) FunctionGetX(ctx context.Context, resourceGroupName, functionName, slotName string) (*armappservice.Site, error) {
	var result *armappservice.Site
	var respErr *azcore.ResponseError
	if slotName == "" {
		resp, err := c.webAppClient.Get(ctx, resourceGroupName, functionName, nil)
		if err != nil {
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return nil, nil
			}
			return nil, err
		}
		result = &resp.Site
	} else {
		resp, err := c.webAppClient.GetSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil {
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return nil, nil
			}
			return nil, err
		}
		result = &resp.Site
	}
	return result, nil
}
func (c *sdkClient) FunctionListSlots(ctx context.Context, resourceGroupName, functionName string) ([]*armappservice.Site, error) {
	result := make([]*armappservice.Site, 0)
	pager := c.webAppClient.NewListSlotsPager(resourceGroupName, functionName, nil)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, resp.Value...)
	}
	return result, nil
}
func (c *sdkClient) FunctionGetPublishCredentialsX(ctx context.Context, resourceGroupName, functionName, slotName string) (*armappservice.User, error) {
	if slotName == "" {
		poller, err := c.webAppClient.BeginListPublishingCredentials(ctx, resourceGroupName, functionName, nil)
		if err != nil {
			return nil, err
		}
		result, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 5 * time.Second})
		if err != nil {
			return nil, err
		}
		return &result.User, nil
	} else {
		poller, err := c.webAppClient.BeginListPublishingCredentialsSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil {
			return nil, err
		}
		result, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 5 * time.Second})
		if err != nil {
			return nil, err
		}
		return &result.User, nil
	}
}
func (c *sdkClient) FunctionGetSCMAllowedX(ctx context.Context, resourceGroupName, functionName, slotName string) (bool, error) {
	if slotName == "" {
		auth, err := c.webAppClient.GetScmAllowed(ctx, resourceGroupName, functionName, nil)
		if err != nil {
			return false, err
		}
		return *auth.Properties.Allow, nil
	} else {
		auth, err := c.webAppClient.GetScmAllowedSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil {
			return false, err
		}
		return *auth.Properties.Allow, nil
	}
}
func (c *sdkClient) FunctionSyncX(ctx context.Context, resourceGroupName, functionName, slotName string) error {
	var respErr *azcore.ResponseError
	if slotName == "" {
		_, err := c.webAppClient.SyncFunctions(ctx, resourceGroupName, functionName, nil)
		if err != nil && errors.As(err, &respErr) && respErr.StatusCode == http.StatusOK { // temporary workaround, server return unexpected 200
			return nil
		}
		return err
	} else {
		_, err := c.webAppClient.SyncFunctionsSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil && errors.As(err, &respErr) && respErr.StatusCode == http.StatusOK {
			return nil
		}
		return err
	}
}
func (c *sdkClient) FunctionListApplicationSettingsX(ctx context.Context, resourceGroupName, functionName, slotName string) (map[string]*string, error) {
	if slotName == "" {
		resp, err := c.webAppClient.ListApplicationSettings(ctx, resourceGroupName, functionName, nil)
		if err != nil {
			return nil, err
		}
		return resp.Properties, nil
	} else {
		resp, err := c.webAppClient.ListApplicationSettingsSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil {
			return nil, err
		}
		return resp.Properties, nil
	}
}
func (c *sdkClient) FunctionUpdateApplicationSettingsX(ctx context.Context, resourceGroupName, functionName, slotName string, props map[string]*string) error {
	if slotName == "" {
		_, err := c.webAppClient.UpdateApplicationSettings(ctx, resourceGroupName, functionName, armappservice.StringDictionary{
			Properties: props,
		}, nil)
		return err
	} else {
		_, err := c.webAppClient.UpdateApplicationSettingsSlot(ctx, resourceGroupName, functionName, slotName, armappservice.StringDictionary{
			Properties: props,
		}, nil)
		return err
	}
}
func (c *sdkClient) StorageBlobSASSign(ctx context.Context, resourceGroupName, blobURL string) (string, error) {
	parsed, err := azblob.ParseURL(blobURL)
	if err != nil {
		return "", err
	}
	if parsed.SAS.Encode() == "" {
		if parsed.ContainerName == "" || parsed.BlobName == "" {
			return "", errors.New("URL needs to follow blob url scheme")
		}
		var accountName string
		if parsed.IPEndpointStyleInfo.AccountName != "" {
			accountName = parsed.IPEndpointStyleInfo.AccountName
		} else {
			accountName = strings.SplitN(parsed.Host, ".", 2)[0]
		}
		keyResp, err := c.accountStorageClient.ListKeys(ctx, resourceGroupName, accountName, nil)
		if err != nil {
			return "", err
		}
		accountKey := *keyResp.Keys[0].Value
		sasCred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return "", err
		}
		sasURLParams, err := sas.BlobSignatureValues{
			Version:       "2024-11-04",
			StartTime:     time.Now().UTC().Add(-5 * time.Minute),
			ExpiryTime:    time.Now().UTC().Add(24 * 7 * 520 * time.Hour),
			ContainerName: parsed.ContainerName,
			BlobName:      parsed.BlobName,
			Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		}.SignWithSharedKey(sasCred)
		if err != nil {
			return "", err
		}
		parsed.SAS = sasURLParams
	}
	return fmt.Sprintf("%s://%s/%s/%s?%s", parsed.Scheme, parsed.Host, parsed.ContainerName, parsed.BlobName, parsed.SAS.Encode()), nil
}
func (c *sdkClient) FunctionKuduDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, packageUri string) error {
	user, err := c.FunctionGetPublishCredentialsX(ctx, resourceGroupName, functionName, slotName)
	if err != nil {
		return err
	}
	packageUri, err = c.StorageBlobSASSign(ctx, resourceGroupName, packageUri)
	if err != nil {
		return err
	}
	body := struct {
		PackageURI string `json:"packageUri"`
	}{
		PackageURI: packageUri,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	deployReq, err := http.NewRequest("POST", *user.Properties.ScmURI+"/api/zipdeploy", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	//TODO: tagging this deployment
	queryParams := deployReq.URL.Query()
	queryParams.Set("isAsync", "true")
	queryParams.Set("Deployer", "pipecd")
	deployReq.URL.RawQuery = queryParams.Encode()
	deployReq.Header.Set("Content-Type", "application/json")
	deployReq.Header.Set("Cache-Control", "no-cache")
	scmAuth, err := c.FunctionGetSCMAllowedX(ctx, resourceGroupName, functionName, slotName)
	if err != nil {
		return err
	}
	var token string
	if !scmAuth {
		resp, err := c.cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{
				"https://management.core.windows.net/.default",
			},
		})
		if err != nil {
			return err
		}
		token = resp.Token
		deployReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	client := &http.Client{}
	deployResp, err := client.Do(deployReq)
	if err != nil {
		return err
	}
	defer deployResp.Body.Close()
	if deployResp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(deployResp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("zip deployment response with status code %d: %s", deployResp.StatusCode, string(body))
	}
	setCookies := deployResp.Cookies() // affinity cookie
	pollLink := deployResp.Header.Get("Location")
	if pollLink == "" {
		return fmt.Errorf("no location header found in deployment response")
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
progressLoop:
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("zip deployment timeout")
		case <-ticker.C:
			progressReq, err := http.NewRequest("GET", pollLink, nil)
			if err != nil {
				return err
			}
			for _, cookie := range setCookies {
				progressReq.AddCookie(cookie)
			}
			if !scmAuth {
				progressReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			} else {
				progressReq.SetBasicAuth(*user.Properties.PublishingUserName, *user.Properties.PublishingPassword)
			}
			progressResp, err := client.Do(progressReq)
			if err != nil {
				return err
			}
			body, err := io.ReadAll(progressResp.Body)
			if err != nil {
				return err
			}
			if progressResp.StatusCode != http.StatusOK && progressResp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("zip deployment status check failed with status code %d, body %s", progressResp.StatusCode, string(body))
			}
			var f struct {
				Status int `json:"status"`
			}
			if err = json.Unmarshal(body, &f); err != nil {
				return err
			}
			if f.Status == 3 {
				return fmt.Errorf("zip deployment failed")
			}
			if f.Status == 4 {
				break progressLoop
			}
		}
	}
	return c.FunctionSyncX(ctx, resourceGroupName, functionName, slotName)
}
func (c *sdkClient) FunctionRunFromPackageDeploymentX(ctx context.Context, resourceGroupName, functionName, slotName, packageUri string) error {
	props, err := c.FunctionListApplicationSettingsX(ctx, resourceGroupName, functionName, slotName)
	if err != nil {
		return err
	}
	packageUri, err = c.StorageBlobSASSign(ctx, resourceGroupName, packageUri)
	if err != nil {
		return err
	}
	props["WEBSITE_MOUNT_ENABLED"] = to.Ptr("1")
	props["WEBSITE_RUN_FROM_PACKAGE"] = to.Ptr(packageUri)
	if err = c.FunctionUpdateApplicationSettingsX(ctx, resourceGroupName, functionName, slotName, props); err != nil {
		return err
	}
	// newly created function need ~30s to get up
	for attempt := 1; attempt <= 3; attempt++ {
		err = c.FunctionSyncX(ctx, resourceGroupName, functionName, slotName)
		if err == nil {
			break
		}
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusBadRequest {
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*20) * time.Second)
			}
			continue
		}
		return err
	}
	return err
}
func (c *sdkClient) FunctionSwapSlotX(ctx context.Context, resourceGroupName, functionName, slotName1, slotName2 string) error {
	if slotName1 == "" && slotName2 == "" {
		return fmt.Errorf("at least one of slot names must be specified")
	}
	if slotName1 == "" || slotName2 == "" {
		poller, err := c.webAppClient.BeginSwapSlotWithProduction(ctx, resourceGroupName, functionName, armappservice.CsmSlotEntity{TargetSlot: to.Ptr(slotName1 + slotName2)}, nil)
		if err != nil {
			return err
		}
		_, err = poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: time.Second * 5})
		return err
	} else {
		poller, err := c.webAppClient.BeginSwapSlot(ctx, resourceGroupName, functionName, slotName1, armappservice.CsmSlotEntity{TargetSlot: to.Ptr(slotName2)}, nil)
		if err != nil {
			return err
		}
		_, err = poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: time.Second * 5})
		return err
	}
}
func (c *sdkClient) FunctionValidate(ctx context.Context, resourceGroupName, functionName string, slotNames []string) (needDeploy bool, err error) {
	_, err = c.resourceGroupClient.Get(ctx, resourceGroupName, nil)
	if err != nil {
		return false, err
	}
	var respErr *azcore.ResponseError
	_, err = c.webAppClient.Get(ctx, resourceGroupName, functionName, nil)
	if err != nil {
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return true, nil
		}
		return false, err
	}
	for _, slotName := range slotNames {
		_, err = c.webAppClient.GetSlot(ctx, resourceGroupName, functionName, slotName, nil)
		if err != nil {
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return true, nil
			}
			return false, err
		}
	}
	return false, nil
}
func (c *sdkClient) parseARMTemplate(appDir string, armTemplate config.AzureDeployTemplate) (*map[string]any, *map[string]*armresources.DeploymentParameter, error) {
	templateContent, err := os.ReadFile(filepath.Join(appDir, armTemplate.DeploymentTemplateFile))
	if err != nil {
		return nil, nil, err
	}
	var templateJson map[string]any
	err = json.Unmarshal(templateContent, &templateJson)
	if err != nil {
		return nil, nil, err
	}
	//TODO: tag the desired resource here
	parameterContent, err := os.ReadFile(filepath.Join(appDir, armTemplate.DeploymentParameterFile))
	if err != nil {
		return nil, nil, err
	}
	var parameterJson map[string]any
	err = json.Unmarshal(parameterContent, &parameterJson)
	if err != nil {
		return nil, nil, err
	}
	rawParams, ok := parameterJson["parameters"].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("parameters field in the parameters file %s is not formated correctly", armTemplate.DeploymentParameterFile)
	}
	// https://learn.microsoft.com/en-us/azure/azure-resource-manager/bicep/parameter-files?tabs=JSON
	deployParameters := make(map[string]*armresources.DeploymentParameter)
	for key, value := range rawParams {
		paramValue, ok := value.(map[string]any)
		if !ok {
			c.logger.Infof("parameters key %s,value %s does not have value formated correctly, skipped", key, value)
			continue
		}
		deployParameters[key] = &armresources.DeploymentParameter{
			Value: paramValue["value"],
		}
	}
	return &templateJson, &deployParameters, nil
}
func (c *sdkClient) DeployARMTemplate(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) error {
	templateJson, deployParameters, err := c.parseARMTemplate(appDir, armTemplate)
	if err != nil {
		return err
	}
	poller, err := c.deploymentClient.BeginCreateOrUpdate(ctx, resourceGroupName, armTemplate.DeploymentName, armresources.Deployment{
		Properties: &armresources.DeploymentProperties{
			Mode:       to.Ptr(armresources.DeploymentModeIncremental),
			Template:   *templateJson,
			Parameters: *deployParameters,
		},
	}, nil)
	if err != nil {
		return err
	}
	resp, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: time.Second * 5})
	if err != nil {
		return err
	}
	propertiesDump, err := json.Marshal(resp.Properties)
	if err == nil {
		c.logger.Infof("ARM template deployed with response: %s", string(propertiesDump))
	}
	return nil
}
func (c *sdkClient) DeployARMTemplateWhatIf(ctx context.Context, resourceGroupName, appDir string, armTemplate config.AzureDeployTemplate) (*armresources.WhatIfOperationProperties, error) {
	templateJson, deployParameters, err := c.parseARMTemplate(appDir, armTemplate)
	if err != nil {
		return nil, err
	}
	poller, err := c.deploymentClient.BeginWhatIf(ctx, resourceGroupName, armTemplate.DeploymentName, armresources.DeploymentWhatIf{
		Properties: &armresources.DeploymentWhatIfProperties{
			Mode:       to.Ptr(armresources.DeploymentModeIncremental),
			Template:   *templateJson,
			Parameters: *deployParameters,
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	resp, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: time.Second * 5})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		errDump, err := json.Marshal(resp.Error)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error whatif: %s", string(errDump))
	}
	return resp.Properties, nil
}
func (c *sdkClient) ContainerAppEnvGet(ctx context.Context, resourceGroupName, environmentName string) (*armappcontainers.ManagedEnvironment, error) {
	result, err := c.managedEnvironmentClient.Get(ctx, resourceGroupName, environmentName, nil)
	if err != nil {
		var fErr *azcore.ResponseError
		if errors.As(err, &fErr) && fErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &result.ManagedEnvironment, nil
}
func (c *sdkClient) ContainerAppEnvCreateOrUpdate(ctx context.Context, resourceGroupName, environmentName string, template *armappcontainers.ManagedEnvironment) error {
	poller, err := c.managedEnvironmentClient.BeginCreateOrUpdate(ctx, resourceGroupName, environmentName, *template, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 5 * time.Second})
	if err != nil {
		return err
	}
	return nil
}
func (c *sdkClient) ContainerAppCreateOrUpdate(ctx context.Context, resourceGroupName, containerAppName string, template *armappcontainers.ContainerApp) error {
	poller, err := c.containerAppClient.BeginCreateOrUpdate(ctx, resourceGroupName, containerAppName, *template, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: 5 * time.Second})
	if err != nil {
		return err
	}
	return nil
}
func NewAzureClient(ctx context.Context, config config.AzureDeployTargetConfig, tags map[string]*string) (AzureClient, error) {
	client := sdkClient{
		tags: tags,
	}
	var err error
	client.cred, err = azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	subscriptionsClient, err := armsubscriptions.NewClient(client.cred, nil)
	if err != nil {
		return nil, err
	}
	_, err = subscriptionsClient.Get(ctx, config.SubscriptionID, nil) // this traps incorrect subscriptionID
	if err != nil {
		return nil, err
	}

	//TODO: only init necessary client, handle subscription issue
	if client.resourceGroupClient, err = armresources.NewResourceGroupsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.accountStorageClient, err = armstorage.NewAccountsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.planClient, err = armappservice.NewPlansClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.webAppClient, err = armappservice.NewWebAppsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.deploymentClient, err = armresources.NewDeploymentsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.managedEnvironmentClient, err = armappcontainers.NewManagedEnvironmentsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.managedClusterClient, err = armcontainerservice.NewManagedClustersClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	if client.containerGroupClient, err = armcontainerinstance.NewContainerGroupsClient(config.SubscriptionID, client.cred, nil); err != nil {
		return nil, err
	}
	return &client, nil
}

const (
	LabelManagedBy   string = "pipecd-dev-managed-by"  // Always be piped.
	LabelPiped       string = "pipecd-dev-piped"       // The id of piped handling this application.
	LabelApplication string = "pipecd-dev-application" // The application this resource belongs to.
	LabelCommitHash  string = "pipecd-dev-commit-hash" // Hash value of the deployed commit.
	ManagedByPiped   string = "piped"
)
