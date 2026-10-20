#!/bin/bash

# Copyright 2025 The PipeCD Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PLUGIN_NAME=$1
CODEOWNERS="${@:2}" # GitHub IDs of each codeowner split by ' '. They must be started by '@'.

if [ -z "$PLUGIN_NAME" -o -z "$CODEOWNERS" ]; then
    echo "Usage: $0 <plugin-name> <codeowner1> <codeowner2> ..."
    exit 1
fi

# if Codeowner does not start by '@', then exit 1.
for codeowner in $CODEOWNERS; do
    if [[ "$codeowner" != @* ]]; then
        echo "CodeOwner must start with '@'. Please check the codeowner: $codeowner"
        exit 1
    fi
done

# 1. Initialize the plugin directory.
PLUGIN_DIR=plugins/$PLUGIN_NAME
mkdir $PLUGIN_DIR
readme=$PLUGIN_DIR/README.md

# 1-1. README
cp hack/init-template/README.md $PLUGIN_DIR/README.md
# title
gsed -i "s|# Plugin Name <!-- Replace the name -->|# ${PLUGIN_NAME} plugin|g" $readme
# issues link
gsed -i "s|{{ISSUES_PLUGIN_NAME}}|${PLUGIN_NAME}|g" $readme
# codeowners link
codeowner_links=""
for codeowner in $CODEOWNERS; do
    codeowner_links="$codeowner_links [$codeowner](https://github.com/$codeowner) "
done
gsed -i "s|@{ACCOUNT}|$codeowner_links|" $readme

## 1-2. Makefile
cp hack/init-template/Makefile $PLUGIN_DIR/Makefile

# 1-3. go.mod
pushd $PLUGIN_DIR
go mod init github.com/pipe-cd/community-plugins/$PLUGIN_DIR
mv go.mod go.mod.tmp
popd

# 1-4. main.go
cat <<EOF > $PLUGIN_DIR/main.go
package main

import (
	"log"

	"github.com/pipe-cd/pipecd/pkg/plugin/sdk/go/plugin"
	"github.com/pipe-cd/community-plugins/plugins/$PLUGIN_NAME/deployment"
)

func main() {
	if err := plugin.Run(&deployment.Plugin{}); err != nil {
		log.Fatalf("failed to run plugin: %v\n", err)
	}
}
EOF

# 1-5. deployment/plugin.go
mkdir -p $PLUGIN_DIR/deployment
cat <<EOF > $PLUGIN_DIR/deployment/plugin.go
package deployment

import (
	"context"

	"github.com/pipe-cd/pipecd/pkg/plugin/sdk/go/plugin"
)

type Plugin struct{}

// Ensure Plugin implements the deployment plugin API.
var _ plugin.DeploymentPlugin = (*Plugin)(nil)

func (p *Plugin) Name() string {
	return "$PLUGIN_NAME"
}

func (p *Plugin) Version() string {
	return "0.1.0"
}
EOF

# 2. Update Issue templates, CODEOWNERS
make sync
