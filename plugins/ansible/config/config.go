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

package config

// AnsiblePluginConfig represents the plugin-level configuration for Ansible.
type AnsiblePluginConfig struct{}

// AnsibleDeployTargetConfig represents the configuration for Ansible deployment targets.
type AnsibleDeployTargetConfig struct {
	AnsiblePath string `json:"ansiblePath,omitempty"`
	Inventory   string `json:"inventory,omitempty"`
	Vault       string `json:"vault,omitempty"`
}

// AnsibleApplicationSpec represents the configuration for Ansible applications.
type AnsibleApplicationSpec struct {
	Name     string                  `json:"name,omitempty"`
	Playbook AnsiblePlaybookManifest `json:"playbook"`
	Pipeline *AnsiblePipelineSpec    `json:"pipeline,omitempty"`
}

// AnsiblePlaybookManifest represents the Ansible playbook configuration.
type AnsiblePlaybookManifest struct {
	Path       string            `json:"path"`
	Inventory  string            `json:"inventory,omitempty"`
	ExtraVars  map[string]string `json:"extraVars,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	SkipTags   []string          `json:"skipTags,omitempty"`
	Limit      string            `json:"limit,omitempty"`
	Verbosity  int               `json:"verbosity,omitempty"`
	CheckMode  bool              `json:"checkMode,omitempty"`
	DiffMode   bool              `json:"diffMode,omitempty"`
	Vault      string            `json:"vault,omitempty"`
	PrivateKey string            `json:"privateKey,omitempty"`
	RemoteUser string            `json:"remoteUser,omitempty"`
	BecomeUser string            `json:"becomeUser,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
}

// AnsiblePipelineSpec represents the pipeline configuration for Ansible applications.
type AnsiblePipelineSpec struct {
	Stages []AnsibleStageSpec `json:"stages,omitempty"`
}

// AnsibleStageSpec represents a stage configuration for Ansible applications.
type AnsibleStageSpec struct {
	Name string                 `json:"name"`
	With map[string]interface{} `json:"with,omitempty"`
}
