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

// Kubectl is a wrapper for kubectl command.
type Sqldef struct {
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
func NewSqldef(logger sdk.StageLogPersister, username, password, host, port, dbName, schemaFilePath, execPath string) *Sqldef {
	return &Sqldef{
		logger:         logger,
		username:       username,
		password:       password,
		host:           host,
		port:           port,
		DBName:         dbName,
		SchemaFilePath: schemaFilePath,
		execPath:       execPath,
	}
}

func (s *Sqldef) ShowCurrentSchema(ctx context.Context) (string, error) {
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

func (s *Sqldef) Execute(ctx context.Context, dryRun, enableDrop bool) error {

	args := []string{
		"-u", s.username,
		"-p", s.password,
		"-h", s.host,
		"-P", s.port,
	}

	if dryRun {
		args = append(args, "--dry-run")
	}
	if enableDrop {
		args = append(args, "--enable-drop")
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

	s.logger.Info("sqldef successfully executed:")
	s.logger.Info(stdout.String())

	return nil
}
