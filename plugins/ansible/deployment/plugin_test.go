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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pipe-cd/community-plugins/plugins/ansible/config"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

func TestPlugin_FetchDefinedStages(t *testing.T) {
	p := &Plugin{}
	stages := p.FetchDefinedStages()

	assert.Len(t, stages, 1)
	assert.Contains(t, stages, "ANSIBLE_SYNC")
}

func TestPlugin_DetermineStrategy(t *testing.T) {
	p := &Plugin{}
	ctx := context.Background()
	cfg := &config.AnsiblePluginConfig{}

	input := &sdk.DetermineStrategyInput[config.AnsibleApplicationSpec]{}

	resp, err := p.DetermineStrategy(ctx, cfg, input)
	require.NoError(t, err)
	assert.Equal(t, sdk.SyncStrategyQuickSync, resp.Strategy)
}

func TestBuildQuickSync(t *testing.T) {
	tests := []struct {
		name         string
		autoRollback bool
		expected     []sdk.QuickSyncStage
	}{
		{
			name:         "without auto rollback",
			autoRollback: false,
			expected: []sdk.QuickSyncStage{
				{
					Name:               "ANSIBLE_SYNC",
					Description:        "Execute Ansible playbook",
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
		{
			name:         "with auto rollback",
			autoRollback: true,
			expected: []sdk.QuickSyncStage{
				{
					Name:               "ANSIBLE_SYNC",
					Description:        "Execute Ansible playbook",
					Metadata:           map[string]string{},
					AvailableOperation: sdk.ManualOperationNone,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildQuickSync(tt.autoRollback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildPipeline(t *testing.T) {
	stages := []sdk.StageConfig{
		{Name: "ANSIBLE_SYNC", Index: 0},
	}

	result := buildPipeline(stages, false)

	expected := []sdk.PipelineStage{
		{
			Name:               "ANSIBLE_SYNC",
			Index:              0,
			Rollback:           false,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		},
	}

	assert.Equal(t, expected, result)
}
