package config

type SqldefDeployTargetConfig struct {
	DbType         string `json:"db_type"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Host           string `json:"host"`
	Port           string `json:"port"`
	DBName         string `json:"db_name"`
	SchemaFilePath string `json:"schema_file_path"`
	DryRun         bool   `json:"dry_run"`
	EnableDrop     bool   `json:"enable_drop"`
}

type SqldefApplicationSpec struct {
}
