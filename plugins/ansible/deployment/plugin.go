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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pipe-cd/community-plugins/plugins/ansible/config"
	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

// Plugin implements sdk.DeploymentPlugin for Ansible.
type Plugin struct{}

var _ sdk.DeploymentPlugin[config.AnsiblePluginConfig, config.AnsibleDeployTargetConfig, config.AnsibleApplicationSpec] = (*Plugin)(nil)

const (
	AnsibleSync Stage = "ANSIBLE_SYNC"
	// TODO: Add rollback stage
	AnsibleRollback Stage = "ANSIBLE_ROLLBACK"
)

type Stage string

var allStages = []string{
	string(AnsibleSync),
}

func (p *Plugin) FetchDefinedStages() []string {
	return allStages
}

func (p *Plugin) BuildPipelineSyncStages(ctx context.Context, cfg *config.AnsiblePluginConfig, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: buildPipeline(input.Request.Stages, input.Request.Rollback),
	}, nil
}

func (p *Plugin) ExecuteStage(ctx context.Context, cfg *config.AnsiblePluginConfig, dts []*sdk.DeployTarget[config.AnsibleDeployTargetConfig], input *sdk.ExecuteStageInput[config.AnsibleApplicationSpec]) (*sdk.ExecuteStageResponse, error) {
	switch input.Request.StageName {
	case string(AnsibleSync):
		return &sdk.ExecuteStageResponse{
			Status: p.executeAnsibleSyncStage(ctx, cfg, dts, input),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported stage: %s", input.Request.StageName)
	}
}

func (p *Plugin) DetermineVersions(ctx context.Context, cfg *config.AnsiblePluginConfig, d *sdk.DetermineVersionsInput[config.AnsibleApplicationSpec]) (*sdk.DetermineVersionsResponse, error) {
	appCfg, err := d.Request.DeploymentSource.AppConfig()
	if err != nil {
		return nil, err
	}

	return &sdk.DetermineVersionsResponse{
		Versions: []sdk.ArtifactVersion{
			{
				Name:    "playbook",
				Version: appCfg.Spec.Playbook.Path,
				URL:     appCfg.Spec.Playbook.Path,
			},
		},
	}, nil
}

func (p *Plugin) DetermineStrategy(ctx context.Context, cfg *config.AnsiblePluginConfig, d *sdk.DetermineStrategyInput[config.AnsibleApplicationSpec]) (*sdk.DetermineStrategyResponse, error) {
	// For now, always use quick sync - pipeline support can be added later
	return &sdk.DetermineStrategyResponse{
		Strategy: sdk.SyncStrategyQuickSync,
	}, nil
}

func (p *Plugin) BuildQuickSyncStages(ctx context.Context, cfg *config.AnsiblePluginConfig, input *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	return &sdk.BuildQuickSyncStagesResponse{
		Stages: buildQuickSync(input.Request.Rollback),
	}, nil
}

func buildQuickSync(autoRollback bool) []sdk.QuickSyncStage {
	out := make([]sdk.QuickSyncStage, 0, 1)
	out = append(out, sdk.QuickSyncStage{
		Name:               string(AnsibleSync),
		Description:        "Execute Ansible playbook",
		Metadata:           map[string]string{},
		AvailableOperation: sdk.ManualOperationNone,
	})
	// Note: rollback is not supported for Ansible plugin
	return out
}

func buildPipeline(stages []sdk.StageConfig, autoRollback bool) []sdk.PipelineStage {
	out := make([]sdk.PipelineStage, 0, len(stages))
	for _, s := range stages {
		out = append(out, sdk.PipelineStage{
			Name:               s.Name,
			Index:              s.Index,
			Rollback:           false,
			Metadata:           map[string]string{},
			AvailableOperation: sdk.ManualOperationNone,
		})
	}

	// Note: rollback is not supported for Ansible plugin
	return out
}

func (p *Plugin) executeAnsibleSyncStage(ctx context.Context, cfg *config.AnsiblePluginConfig, dts []*sdk.DeployTarget[config.AnsibleDeployTargetConfig], input *sdk.ExecuteStageInput[config.AnsibleApplicationSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()

	appCfg, err := input.Request.TargetDeploymentSource.AppConfig()
	if err != nil {
		lp.Errorf("Failed to get app config: %v", err)
		return sdk.StageStatusFailure
	}

	// Get deploy target config (use first one if multiple)
	var dtConfig *config.AnsibleDeployTargetConfig
	if len(dts) > 0 {
		dtConfig = &dts[0].Config
	} else {
		dtConfig = &config.AnsibleDeployTargetConfig{}
	}

	return p.executeAnsiblePlaybook(ctx, cfg, dtConfig, &appCfg.Spec.Playbook, input)
}

func (p *Plugin) executeAnsiblePlaybook(ctx context.Context, cfg *config.AnsiblePluginConfig, dtConfig *config.AnsibleDeployTargetConfig, playbookConfig *config.AnsiblePlaybookManifest, input *sdk.ExecuteStageInput[config.AnsibleApplicationSpec]) sdk.StageStatus {
	lp := input.Client.LogPersister()

	// Log deployment start
	lp.Infof("🚀 Starting Ansible deployment")
	lp.Infof("📁 Playbook: %s", playbookConfig.Path)
	lp.Infof("🎯 Target directory: %s", input.Request.TargetDeploymentSource.ApplicationDirectory)

	ansiblePath := dtConfig.AnsiblePath
	if ansiblePath == "" {
		ansiblePath = "ansible-playbook"
	}
	lp.Infof("⚙️  Ansible executable: %s", ansiblePath)

	playbookPath := filepath.Join(input.Request.TargetDeploymentSource.ApplicationDirectory, playbookConfig.Path)

	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		lp.Errorf("❌ Playbook file does not exist: %s", playbookPath)

		// Debug: List directory contents to help diagnose
		if dirContent, err := os.ReadDir(input.Request.TargetDeploymentSource.ApplicationDirectory); err == nil {
			lp.Infof("📂 ApplicationDirectory contents:")
			for _, entry := range dirContent {
				lp.Infof("  - %s (isDir: %v)", entry.Name(), entry.IsDir())
			}
		}

		return sdk.StageStatusFailure
	}

	lp.Infof("✅ Playbook file found: %s", playbookPath)

	// Build command arguments
	lp.Infof("🔧 Building ansible-playbook arguments...")
	args := []string{playbookPath}

	// Inventory
	inventory := playbookConfig.Inventory
	if inventory == "" {
		inventory = dtConfig.Inventory
	}
	if inventory != "" {
		inventoryPath := filepath.Join(input.Request.TargetDeploymentSource.ApplicationDirectory, inventory)
		args = append(args, "-i", inventoryPath)
		lp.Infof("📋 Using inventory: %s", inventoryPath)
	} else {
		lp.Infof("📋 No inventory specified, using default")
	}

	// Extra variables
	if len(playbookConfig.ExtraVars) > 0 {
		extraVars := make([]string, 0, len(playbookConfig.ExtraVars))
		for k, v := range playbookConfig.ExtraVars {
			extraVars = append(extraVars, fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, "--extra-vars", strings.Join(extraVars, " "))
		lp.Infof("🔧 Extra variables: %v", playbookConfig.ExtraVars)
	}

	// Tags
	if len(playbookConfig.Tags) > 0 {
		args = append(args, "--tags", strings.Join(playbookConfig.Tags, ","))
		lp.Infof("🏷️  Running with tags: %v", playbookConfig.Tags)
	}

	// Skip tags
	if len(playbookConfig.SkipTags) > 0 {
		args = append(args, "--skip-tags", strings.Join(playbookConfig.SkipTags, ","))
		lp.Infof("🚫 Skipping tags: %v", playbookConfig.SkipTags)
	}

	// Limit
	if playbookConfig.Limit != "" {
		args = append(args, "--limit", playbookConfig.Limit)
		lp.Infof("🎯 Limiting to hosts: %s", playbookConfig.Limit)
	}

	// Verbosity
	if playbookConfig.Verbosity > 0 {
		verbosity := strings.Repeat("v", playbookConfig.Verbosity)
		args = append(args, fmt.Sprintf("-%s", verbosity))
		lp.Infof("📢 Verbosity level: %d", playbookConfig.Verbosity)
	}

	// Check mode
	if playbookConfig.CheckMode {
		args = append(args, "--check")
		lp.Infof("🔍 Running in check mode (dry-run)")
	}

	// Diff mode
	if playbookConfig.DiffMode {
		args = append(args, "--diff")
		lp.Infof("📊 Diff mode enabled")
	}

	// Vault
	vault := playbookConfig.Vault
	if vault == "" {
		vault = dtConfig.Vault
	}
	if vault != "" {
		vaultPath := filepath.Join(input.Request.TargetDeploymentSource.ApplicationDirectory, vault)
		args = append(args, "--vault-password-file", vaultPath)
		lp.Infof("🔐 Using vault password file: %s", vaultPath)
	}

	// Private key
	if playbookConfig.PrivateKey != "" {
		keyPath := filepath.Join(input.Request.TargetDeploymentSource.ApplicationDirectory, playbookConfig.PrivateKey)
		args = append(args, "--private-key", keyPath)
		lp.Infof("🔑 Using private key: %s", keyPath)
	}

	// Remote user
	if playbookConfig.RemoteUser != "" {
		args = append(args, "--user", playbookConfig.RemoteUser)
		lp.Infof("👤 Remote user: %s", playbookConfig.RemoteUser)
	}

	// Become user
	if playbookConfig.BecomeUser != "" {
		args = append(args, "--become", "--become-user", playbookConfig.BecomeUser)
		lp.Infof("🔓 Become user: %s", playbookConfig.BecomeUser)
	}

	lp.Infof("🚀 Executing ansible-playbook command: %s %s", ansiblePath, strings.Join(args, " "))
	lp.Infof("📁 Working directory: %s", input.Request.TargetDeploymentSource.ApplicationDirectory)

	// Track execution timing
	startTime := time.Now()
	lp.Infof("⏱️  Execution started at: %s", startTime.Format("2006-01-02 15:04:05"))

	// Apply timeout if specified
	cmdCtx := ctx
	if playbookConfig.Timeout > 0 {
		var cancel context.CancelFunc
		cmdCtx, cancel = context.WithTimeout(ctx, time.Duration(playbookConfig.Timeout)*time.Second)
		defer cancel()
		lp.Infof("⏰ Timeout set to: %d seconds", playbookConfig.Timeout)
	}

	cmd := exec.CommandContext(cmdCtx, ansiblePath, args...)
	cmd.Dir = input.Request.TargetDeploymentSource.ApplicationDirectory
	cmd.Env = os.Environ()
	cmd.Stdout = lp
	cmd.Stderr = lp

	lp.Infof("📜 Ansible playbook output:")
	lp.Infof("=" + strings.Repeat("=", 80))

	if err := cmd.Run(); err != nil {
		duration := time.Since(startTime)
		lp.Errorf("❌ Failed to execute ansible-playbook: %v", err)
		lp.Errorf("💥 Exit code: %v", cmd.ProcessState.ExitCode())
		lp.Errorf("⏱️  Execution duration: %v", duration)
		lp.Errorf("=" + strings.Repeat("=", 80))
		return sdk.StageStatusFailure
	}

	duration := time.Since(startTime)
	lp.Infof("=" + strings.Repeat("=", 80))
	lp.Infof("✅ Ansible playbook executed successfully!")
	lp.Infof("⏱️  Total execution time: %v", duration)
	lp.Infof("🎉 Deployment completed at: %s", time.Now().Format("2006-01-02 15:04:05"))
	return sdk.StageStatusSuccess
}
