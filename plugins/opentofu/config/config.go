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

// Config represents the plugin-scoped configuration.
type Config struct{}

// DeployTargetConfig represents the deploy-target-scoped configuration.
type DeployTargetConfig struct {
	// List of variables that will be set directly on opentofu commands with "-var" flag.
	// The variable must be formatted by "key=value" as below:
	// "image_id=ami-abc123"
	// 'image_id_list=["ami-abc123","ami-def456"]'
	// 'image_id_map={"us-east-1":"ami-abc123","us-east-2":"ami-def456"}'
	Vars []string `json:"vars,omitempty"`
	// Enable drift detection.
	// TODO: This is a temporary option because  drift detection is buggy and has performance issues. This will be possibly removed in the future release.
	DriftDetectionEnabled *bool `json:"driftDetectionEnabled" default:"true"`
}

// ApplicationConfigSpec represents the application-scoped plugin config.
type ApplicationConfigSpec struct {
	// The opentofu workspace name.
	// Empty means "default" workspace.
	Workspace string `json:"workspace,omitempty"`
	// The version of opentofu that should be used.
	// Empty means the pre-installed version will be used.
	OpenTofuVersion string `json:"openTofuVersion,omitempty"`
	// List of variables that will be set directly on opentofu commands with "-var" flag.
	// The variable must be formatted by "key=value" as below:
	// "image_id=ami-abc123"
	// 'image_id_list=["ami-abc123","ami-def456"]'
	// 'image_id_map={"us-east-1":"ami-abc123","us-east-2":"ami-def456"}'
	Vars []string `json:"vars,omitempty"`
	// List of variable files that will be set on opentofu commands with "-var-file" flag.
	VarFiles []string `json:"varFiles,omitempty"`
	// List of additional flags will be used while executing opentofu commands.
	CommandFlags OpenTofuCommandFlags `json:"commandFlags"`
	// List of additional environment variables will be used while executing opentofu commands.
	CommandEnvs OpenTofuCommandEnvs `json:"commandEnvs"`
}

// OpenTofuPlanStageOptions contains all configurable values for an OPENTOFU_PLAN stage.
type OpenTofuPlanStageOptions struct {
	// Exit the pipeline if the result is "No Changes" with success status.
	ExitOnNoChanges bool `json:"exitOnNoChanges"`
}

// OpenTofuApplyStageOptions contains all configurable values for an OPENTOFU_APPLY stage.
type OpenTofuApplyStageOptions struct {
	// AutoApprove enables -auto-approve flag for tofu apply.
	AutoApprove bool `json:"autoApprove"`
}

// OpenTofuCommandFlags contains all additional flags that will be used while executing opentofu commands.
type OpenTofuCommandFlags struct {
	Shared []string `json:"shared"`
	Init   []string `json:"init"`
	Plan   []string `json:"plan"`
	Apply  []string `json:"apply"`
}

// OpenTofuCommandEnvs contains all additional environment variables that will be used while executing opentofu commands.
type OpenTofuCommandEnvs struct {
	Shared []string `json:"shared"`
	Init   []string `json:"init"`
	Plan   []string `json:"plan"`
	Apply  []string `json:"apply"`
}

func (s *ApplicationConfigSpec) Validate() error {
	// TODO: Validate ApplicationConfigSpec fields.
	return nil
}
