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

// Package toolregistry installs and manages the needed tools
// such as kubectl, helm... for executing tasks in pipeline.
package toolregistry

import (
	"cmp"
	"context"
	"fmt"

	"github.com/pipe-cd/community-plugins/plugins/sqldef/config"
)

const (
	defaultSqldefVersion = "2.0.4"
)

type client interface {
	InstallTool(ctx context.Context, name, version, script string) (string, error)
}

func NewRegistry(client client) *Registry {
	return &Registry{
		client: client,
	}
}

// Registry provides functions to get path to the needed tools.
type Registry struct {
	client client
}

// DownloadBinary downloads the appropriate sqldef binary based on the database type.
// Currently only supports MySQL, but can be extended for other database types.
func (r *Registry) DownloadBinary(ctx context.Context, dbType config.DBType, version string) (string, error) {
	switch dbType {
	case config.DBTypeMySQL:
		return r.client.InstallTool(ctx, "mysqldef", cmp.Or(version, defaultSqldefVersion), MysqldefInstallScript)
	default:
		return "", fmt.Errorf("unsupported database type: %s, currently only support: mysql", dbType)
	}
}
