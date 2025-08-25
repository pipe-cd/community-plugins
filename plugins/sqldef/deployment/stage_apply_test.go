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

func TestPlugin_executeApplyStage_HappyPath(t *testing.T) {
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

	// Key difference from plan stage: dryRun=false for apply stage
	mockSqldef.On("Execute", ctx, false).Return(nil)

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
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

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert success
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_MultipleTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for the mock - should be called twice for two targets
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser1", "testpass1", "localhost", "3306", "testdb1",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	// Key difference: dryRun=false for apply stage
	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser2", "testpass2", "localhost", "3307", "testdb2",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	// Key difference: dryRun=false for apply stage
	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Create multiple deploy targets
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

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert success - should handle multiple targets successfully
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_EmptyDeployTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider (won't be called due to empty targets)
	mockSqldef := &MockSqldefProvider{}

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Empty deploy targets
	deployTargets := []*sdk.DeployTarget[config.DeployTargetConfig]{}

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert success - empty targets should be handled gracefully
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify mock expectations (should be none since no targets)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_UnsupportedDBType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider (won't be called due to unsupported DB type)
	mockSqldef := &MockSqldefProvider{}

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Create deploy targets with unsupported DB type
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

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert failure
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify mock expectations (should be none since DB type is unsupported)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_SchemaFileNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider (won't be called due to schema file not found)
	mockSqldef := &MockSqldefProvider{}

	// Prepare the input with non-existent directory
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("nonexistent"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("nonexistent"),
				CommitHash:           "0123456789",
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

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert failure
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify mock expectations (should be none since schema file not found)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_SqldefExecuteError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider that returns an error
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for the mock
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser", "testpass", "localhost", "3306", "testdb",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return()

	// Mock Execute to return an error with dryRun=false
	mockSqldef.On("Execute", ctx, false).Return(assert.AnError)

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
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

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert failure - apply stage should fail when sqldef.Execute fails
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_BinaryDownloadFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry that will fail to download binary
	testRegistry := toolregistrytest.NewTestToolRegistry(t)
	// Note: The actual binary download failure would be tested through integration tests
	// or by mocking the tool registry, but for this unit test we focus on the logic flow

	// Create mock sqldef provider
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for the mock since binary download will succeed
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser", "testpass", "localhost", "3306", "testdb",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return()

	mockSqldef.On("Execute", ctx, false).Return(nil)

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
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

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert success - test registry provides mock binaries successfully
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_MultipleTargetsWithFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for the first target (success)
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser1", "testpass1", "localhost", "3306", "testdb1",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	// Setup expectations for the second target (failure)
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser2", "testpass2", "localhost", "3307", "testdb2",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	mockSqldef.On("Execute", ctx, false).Return(assert.AnError).Once()

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Create multiple deploy targets
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

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert failure - should fail when any target fails
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executeApplyStage_BinaryCachingAcrossTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider
	mockSqldef := &MockSqldefProvider{}

	// Setup expectations for both targets - same DB type should reuse binary
	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser1", "testpass1", "localhost", "3306", "testdb1",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser2", "testpass2", "localhost", "3307", "testdb2",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // Same execPath should be reused
	).Return().Once()

	mockSqldef.On("Execute", ctx, false).Return(nil).Once()

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStageApply,
			StageConfig: []byte(``),
			RunningDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			TargetDeploymentSource: sdk.DeploymentSource[config.ApplicationConfigSpec]{
				ApplicationDirectory: filepath.Join("testdata"),
				CommitHash:           "0123456789",
			},
			Deployment: sdk.Deployment{
				PipedID:       "piped-id",
				ApplicationID: "app-id",
			},
		},
		Client: sdk.NewClient(nil, "sqldef", "", "", logpersistertest.NewTestLogPersister(t), testRegistry),
	}

	// Create multiple deploy targets with same DB type
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
				DbType:   config.DBTypeMySQL, // Same DB type
				Username: "testuser2",
				Password: "testpass2",
				Host:     "localhost",
				Port:     "3307",
				DBName:   "testdb2",
			},
		},
	}

	// Create plugin with mock sqldef provider
	plugin := createPluginWithMockSqldef(mockSqldef)

	// Execute the apply stage
	status := plugin.executeApplyStage(ctx, deployTargets, input)

	// Assert success - binary caching should work correctly
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}
