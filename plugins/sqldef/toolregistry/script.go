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

const MysqldefInstallScript = `
cd {{ .TmpDir }}
curl -LO https://github.com/sqldef/sqldef/releases/download/v{{ .Version }}/mysqldef_{{ .Os }}_{{ .Arch }}.zip

unzip mysqldef_{{ .Os }}_{{ .Arch }}.zip
chmod +x mysqldef
cp mysqldef {{ .OutPath }}
`
