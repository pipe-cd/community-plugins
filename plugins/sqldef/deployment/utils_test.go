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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFirstSQLFile(t *testing.T) {
	t.Parallel()

	t.Run("finds the sql file in a flat directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		want := filepath.Join(dir, "schema.sql")
		require.NoError(t, os.WriteFile(want, []byte("CREATE TABLE t (id INT);"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not sql"), 0o600))

		got, err := findFirstSQLFile(dir)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("finds the sql file in a nested directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		nested := filepath.Join(dir, "nested")
		require.NoError(t, os.Mkdir(nested, 0o755))
		want := filepath.Join(nested, "schema.sql")
		require.NoError(t, os.WriteFile(want, []byte("CREATE TABLE t (id INT);"), 0o600))

		got, err := findFirstSQLFile(dir)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("returns an error when the directory has no sql file (empty input)", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not sql"), 0o600))

		got, err := findFirstSQLFile(dir)

		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, got)
	})

	t.Run("returns an error when the directory is completely empty", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		got, err := findFirstSQLFile(dir)

		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, got)
	})

	t.Run("returns an error when the directory does not exist", func(t *testing.T) {
		t.Parallel()

		got, err := findFirstSQLFile(filepath.Join(t.TempDir(), "does-not-exist"))

		assert.Error(t, err)
		assert.Empty(t, got)
	})
}
