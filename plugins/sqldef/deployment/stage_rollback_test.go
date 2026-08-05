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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
)

func TestPlugin_executeRollbackStage_HappyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for the mock
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser", "testpass", "localhost", "3306", "testdb",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return()

	// Rollback stage uses dryRun=false, same as apply
	mockSqldef.On("Execute", ctx, false).Return(nil)

	// Prepare the input. The rollback stage reads from RunningDeploymentSource.
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "testdata",
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "nonexistent",
				CommitHash:           "9876543210",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Create deploy targets
	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
		{
			Name: "test-mysql",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypeMySQL,
				Username: "testuser",
				Password: "testpass",
				Host:     "localhost",
				Port:     "3306",
				DBName:   "testdb",
			},
		},
	}

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the rollback stage
	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	// Assert success
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeRollbackStage_MultipleTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	mockSqldef := &MockSqldefProvider{}

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser1", "testpass1", "localhost", "3306", "testdb1",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"),
	).Return().Once()
	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser2", "testpass2", "localhost", "3307", "testdb2",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"),
	).Return().Once()
	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "testdata",
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
		{
			Name: "test-mysql-1",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypeMySQL,
				Username: "testuser1",
				Password: "testpass1",
				Host:     "localhost",
				Port:     "3306",
				DBName:   "testdb1",
			},
		},
		{
			Name: "test-mysql-2",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypeMySQL,
				Username: "testuser2",
				Password: "testpass2",
				Host:     "localhost",
				Port:     "3307",
				DBName:   "testdb2",
			},
		},
	}

	plugin := createPluginWithMockSqldef(mockSqldef)

	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	assert.Equal(t, sdk.StageStatusSuccess, status)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeRollbackStage_EmptyDeployTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	mockSqldef := &MockSqldefProvider{}

	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "testdata",
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{}

	plugin := createPluginWithMockSqldef(mockSqldef)

	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	// Empty targets should be handled gracefully
	assert.Equal(t, sdk.StageStatusSuccess, status)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeRollbackStage_UnsupportedDBType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	mockSqldef := &MockSqldefProvider{}

	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "testdata",
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Rollback with an unsupported / invalid DB type in the config
	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
		{
			Name: "test-postgres",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypePostgres, // Unsupported type
				Username: "testuser",
				Password: "testpass",
				Host:     "localhost",
				Port:     "5432",
				DBName:   "testdb",
			},
		},
	}

	plugin := createPluginWithMockSqldef(mockSqldef)

	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	assert.Equal(t, sdk.StageStatusFailure, status)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeRollbackStage_SchemaFileNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	mockSqldef := &MockSqldefProvider{}

	// The running deployment source points at a directory with no .sql file.
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "nonexistent",
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
		{
			Name: "test-mysql",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypeMySQL,
				Username: "testuser",
				Password: "testpass",
				Host:     "localhost",
				Port:     "3306",
				DBName:   "testdb",
			},
		},
	}

	plugin := createPluginWithMockSqldef(mockSqldef)

	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	assert.Equal(t, sdk.StageStatusFailure, status)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeRollbackStage_SqldefExecuteError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	mockSqldef := &MockSqldefProvider{}

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser", "testpass", "localhost", "3306", "testdb",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"),
	).Return()

	mockSqldef.On("Execute", ctx, false).Return(assert.AnError)

	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageRollback,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: "testdata",
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{
		{
			Name: "test-mysql",
			Config: config.DeployTargetConfig{
				DbType:   config.DBTypeMySQL,
				Username: "testuser",
				Password: "testpass",
				Host:     "localhost",
				Port:     "3306",
				DBName:   "testdb",
			},
		},
	}

	plugin := createPluginWithMockSqldef(mockSqldef)

	status := plugin.executeRollbackStage(ctx, deployTargets, input)

	assert.Equal(t, sdk.StageStatusFailure, status)
	mockSqldef.AssertExpectations(t)
}
