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

package toolregistry

import (
	"context"
	"testing"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_DownloadBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dbType      config.DBType
		version     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "MySQL_with_specific_version",
			dbType:      config.DBTypeMySQL,
			version:     "2.0.4",
			expectError: false,
		},
		{
			name:        "MySQL_with_empty_version_should_use_default",
			dbType:      config.DBTypeMySQL,
			version:     "",
			expectError: false,
		},
		{
			name:        "PostgreSQL_unsupported",
			dbType:      config.DBTypePostgres,
			version:     "2.0.4",
			expectError: true,
			errorMsg:    "unsupported database type: psql, currently only support: mysql",
		},
		{
			name:        "SQLite_unsupported",
			dbType:      config.DBTypeSQLite,
			version:     "2.0.4",
			expectError: true,
			errorMsg:    "unsupported database type: sqlite, currently only support: mysql",
		},
		{
			name:        "MSSQL_unsupported",
			dbType:      config.DBTypeMSSQL,
			version:     "2.0.4",
			expectError: true,
			errorMsg:    "unsupported database type: mssql, currently only support: mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := toolregistrytest.NewTestToolRegistry(t)
			r := NewRegistry(c)

			path, err := r.DownloadBinary(context.Background(), tt.dbType, tt.version)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				require.NotEmpty(t, path)

				// For MySQL, verify the path contains the expected binary name
				if tt.dbType == config.DBTypeMySQL {
					assert.Contains(t, path, "mysqldef", "Expected mysqldef binary name in path")

					// If version was provided, check it's in the path
					if tt.version != "" {
						assert.Contains(t, path, tt.version, "Expected version in path")
					} else {
						// If no version provided, should use default
						assert.Contains(t, path, defaultSqldefVersion, "Expected default version in path")
					}
				}
				// TODO: Add more dbType cases when supported
			}
		})
	}
}
