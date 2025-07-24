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
	"fmt"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
	"os"
	"os/exec"
)

type SqldefProvider interface {
	Init(logger sdk.StageLogPersister, username, password, host, port, dbName, schemaFilePath, execPath string)
	ShowCurrentSchema(ctx context.Context) (string, error)
	Execute(ctx context.Context, dryRun bool) error
}

type SqldefProviderImpl struct {
	logger         sdk.StageLogPersister
	username       string
	password       string
	host           string
	port           string
	DBName         string
	SchemaFilePath string
	execPath       string
}

// NewSqldef creates a new Sqldef instance.
func (s *SqldefProviderImpl) Init(logger sdk.StageLogPersister, username, password, host, port, dbName, schemaFilePath, execPath string) {
	s.logger = logger
	s.username = username
	s.password = password
	s.host = host
	s.port = port
	s.DBName = dbName
	s.SchemaFilePath = schemaFilePath
	s.execPath = execPath
}

func (s *SqldefProviderImpl) ShowCurrentSchema(ctx context.Context) (string, error) {
	args := []string{
		"-u", s.username,
		"-p", s.password,
		"-h", s.host,
		"-P", s.port,
		"--export",
		s.DBName,
	}

	cmd := exec.CommandContext(ctx, s.execPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run mysqldef: %w, stderr: %s", err, errBuf.String())
	}

	return outBuf.String(), nil
}

func (s *SqldefProviderImpl) Execute(ctx context.Context, dryRun bool) error {
	args := []string{
		"-u", s.username,
		"-p", s.password,
		"-h", s.host,
		"-P", s.port,
		"--enable-drop",
	}

	if dryRun {
		args = append(args, "--dry-run")
	}

	args = append(args, s.DBName)

	file, err := os.Open(s.SchemaFilePath)
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}
	defer file.Close()

	cmd := exec.CommandContext(ctx, s.execPath, args...)
	cmd.Stdin = file

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldef execution failed: %w\nstderr: %s", err, stderr.String())
	}

	if dryRun {
		s.logger.Info("Dry run mode: the following SQL statements would be executed:")
	} else {
		s.logger.Info("sqldef executed successfully. Output:")
	}
	s.logger.Info(stdout.String())

	return nil
}
