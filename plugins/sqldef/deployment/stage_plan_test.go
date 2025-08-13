package deployment

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	"github.com/pipe-cd/community-plugins/plugins/sqldef/provider"
)

// MockSqldefProvider is a mock implementation of provider.SqldefProvider
type MockSqldefProvider struct {
	mock.Mock
}

func (m *MockSqldefProvider) Init(logger sdk.StageLogPersister, username, password, host, port, dbName, schemaFilePath, execPath string) {
	m.Called(logger, username, password, host, port, dbName, schemaFilePath, execPath)
}

func (m *MockSqldefProvider) ShowCurrentSchema(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockSqldefProvider) Execute(ctx context.Context, dryRun bool) error {
	args := m.Called(ctx, dryRun)
	return args.Error(0)
}

// createPluginWithMockSqldef creates a Plugin instance with a mock sqldef provider
func createPluginWithMockSqldef(mockProvider provider.SqldefProvider) *Plugin {
	return &Plugin{
		Sqldef: mockProvider,
	}
}

func TestPlugin_executePlanStage_HappyPath(t *testing.T) {
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

	mockSqldef.On("Execute", ctx, true).Return(nil)

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStagePlan,
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

	// Execute the plan stage
	status := plugin.executePlanStage(ctx, deployTargets, input)

	// Assert success
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executePlanStage_EmptySchemaFileName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider (won't be called due to early validation failure)
	mockSqldef := &MockSqldefProvider{}

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStagePlan,
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

	// Create deploy targets with empty schema file name
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

	mockSqldef.On("Init",
		mock.Anything, // log persister
		"testuser",
		"testpass",
		"localhost",
		"3306",
		"testdb",
		"testdata/schema.sql",
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return(nil)

	mockSqldef.On("Execute", mock.Anything, true).Return(errors.New("execute failed."))

	// Execute the plan stage
	status := plugin.executePlanStage(ctx, deployTargets, input)

	// Assert failure
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify mock expectations (should be none since validation fails early)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executePlanStage_UnsupportedDBType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Initialize tool registry
	testRegistry := toolregistrytest.NewTestToolRegistry(t)

	// Create mock sqldef provider (won't be called due to unsupported DB type)
	mockSqldef := &MockSqldefProvider{}

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStagePlan,
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

	// Execute the plan stage
	status := plugin.executePlanStage(ctx, deployTargets, input)

	// Assert failure
	assert.Equal(t, sdk.StageStatusFailure, status)

	// Verify mock expectations (should be none since DB type is unsupported)
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executePlanStage_SqldefExecuteError(t *testing.T) {
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

	// Mock Execute to return an error
	mockSqldef.On("Execute", ctx, true).Return(nil)

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStagePlan,
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

	// Execute the plan stage
	status := plugin.executePlanStage(ctx, deployTargets, input)

	// Assert success (function continues even if sqldef.Execute fails)
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}

func TestPlugin_executePlanStage_MultipleTargets(t *testing.T) {
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

	mockSqldef.On("Execute", ctx, true).Return(nil).Once()

	mockSqldef.On("Init",
		mock.AnythingOfType("logpersistertest.TestLogPersister"),
		"testuser2", "testpass2", "localhost", "3307", "testdb2",
		filepath.Join("testdata", "schema.sql"),
		mock.AnythingOfType("string"), // execPath from tool registry
	).Return().Once()

	mockSqldef.On("Execute", ctx, true).Return(nil).Once()

	// Prepare the input
	input := &sdk.ExecuteStageInput[config.ApplicationConfigSpec]{
		Request: sdk.ExecuteStageRequest[config.ApplicationConfigSpec]{
			StageName:   sqldefStagePlan,
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

	// Execute the plan stage
	status := plugin.executePlanStage(ctx, deployTargets, input)

	// Assert success - should handle multiple targets successfully
	assert.Equal(t, sdk.StageStatusSuccess, status)

	// Verify all mock expectations were met
	mockSqldef.AssertExpectations(t)
}
