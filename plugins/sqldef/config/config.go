package config

type DBType string

const (
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "psql"
	DBTypeSQLite   DBType = "sqlite"
	DBTypeMSSQL    DBType = "mssql"
)

type Config struct{}

type DeployTargetConfig struct {
	// DbType include: mysql, psql, sqlite, and mssql
	DbType DBType `json:"db_type"`
	// DB connection info
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	DBName   string `json:"db_name"`
}

type ApplicationConfigSpec struct {
}
