package config

type Config struct{}

type DeployTargetConfig struct {
	// DbType include: mysql, psql, sqlite, and mssql
	DbType string `json:"db_type"`
	// DB connection info
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	DBName   string `json:"db_name"`
	// sqldef input parameters
	SchemaFileName string `json:"schema_file_name"`
	DryRun         bool   `json:"dry_run"`
	EnableDrop     bool   `json:"enable_drop"`
}

type ApplicationConfigSpec struct {
}
