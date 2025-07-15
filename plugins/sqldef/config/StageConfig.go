package config

import (
	"encoding/json"
	"fmt"
)

type SqldefSyncStageConfig struct {
	EnableDrop bool `json:"enable_drop,omitempty"`
}

func DecodeConfig(data []byte) (SqldefSyncStageConfig, error) {
	var config SqldefSyncStageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return SqldefSyncStageConfig{}, fmt.Errorf("failed to unmarshal the config: %w", err)
	}
	return config, nil
}
