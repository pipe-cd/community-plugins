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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pipe-cd/piped-plugin-sdk-go/logpersister/logpersistertest"
)

// writeFakeBinary writes an executable shell script standing in for the
// sqldef binary, so tests can exercise SqldefProviderImpl without shelling
// out to a real tool or network resource.
func writeFakeBinary(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fake-sqldef.sh")
	script := "#!/bin/sh\n" + body
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

// writeSchemaFile writes a schema file and returns its path.
func writeSchemaFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "schema.sql")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestSqldefProviderImpl_ShowCurrentSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scriptBody  string
		expectError bool
		expectedOut string
	}{
		{
			name:        "success",
			scriptBody:  "echo 'CREATE TABLE users (id INT);'\nexit 0\n",
			expectError: false,
			expectedOut: "CREATE TABLE users (id INT);\n",
		},
		{
			name:        "command failure",
			scriptBody:  "echo 'boom' >&2\nexit 1\n",
			expectError: true,
			expectedOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			execPath := writeFakeBinary(t, tt.scriptBody)

			s := &SqldefProviderImpl{}
			s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb", "", execPath)

			out, err := s.ShowCurrentSchema(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, out)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOut, out)
			}
		})
	}
}

func TestSqldefProviderImpl_Execute(t *testing.T) {
	t.Parallel()

	t.Run("success with dry run", func(t *testing.T) {
		t.Parallel()

		execPath := writeFakeBinary(t, "cat > /dev/null\nexit 0\n")
		schemaPath := writeSchemaFile(t, "CREATE TABLE users (id INT);")

		s := &SqldefProviderImpl{}
		s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb", schemaPath, execPath)

		err := s.Execute(context.Background(), true)

		assert.NoError(t, err)
	})

	t.Run("success without dry run", func(t *testing.T) {
		t.Parallel()

		execPath := writeFakeBinary(t, "cat > /dev/null\nexit 0\n")
		schemaPath := writeSchemaFile(t, "CREATE TABLE users (id INT);")

		s := &SqldefProviderImpl{}
		s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb", schemaPath, execPath)

		err := s.Execute(context.Background(), false)

		assert.NoError(t, err)
	})

	t.Run("command failure", func(t *testing.T) {
		t.Parallel()

		execPath := writeFakeBinary(t, "cat > /dev/null\necho 'syntax error' >&2\nexit 1\n")
		schemaPath := writeSchemaFile(t, "CREATE TABLE users (id INT);")

		s := &SqldefProviderImpl{}
		s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb", schemaPath, execPath)

		err := s.Execute(context.Background(), false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("schema file not found", func(t *testing.T) {
		t.Parallel()

		execPath := writeFakeBinary(t, "exit 0\n")

		s := &SqldefProviderImpl{}
		s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb",
			filepath.Join(t.TempDir(), "does-not-exist.sql"), execPath)

		err := s.Execute(context.Background(), false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open schema file")
	})

	t.Run("empty schema file", func(t *testing.T) {
		t.Parallel()

		execPath := writeFakeBinary(t, "cat > /dev/null\nexit 0\n")
		schemaPath := writeSchemaFile(t, "")

		s := &SqldefProviderImpl{}
		s.Init(logpersistertest.NewTestLogPersister(t), "user", "pass", "localhost", "3306", "testdb", schemaPath, execPath)

		err := s.Execute(context.Background(), false)

		assert.NoError(t, err)
	})
}
