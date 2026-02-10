package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	SeaweedFS SeaweedFSConfig `mapstructure:"seaweedfs"`
	Database  DatabaseConfig  `mapstructure:"database"`
	User      UserConfig      `mapstructure:"user"`
}

type SeaweedFSConfig struct {
	S3Endpoint string `mapstructure:"s3_endpoint"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	BucketName string `mapstructure:"bucket_name"`
	UseSSL     bool   `mapstructure:"use_ssl"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type UserConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Username string `mapstructure:"username"`
}

func Load() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("VAULT")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createDefaultConfig(configDir); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("seaweedfs.s3_endpoint", "http://localhost:8333")
	viper.SetDefault("seaweedfs.bucket_name", "code-vault")
	viper.SetDefault("seaweedfs.use_ssl", false)

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "vault_user")
	viper.SetDefault("database.dbname", "code_vault")
	viper.SetDefault("database.sslmode", "disable")
}

func createDefaultConfig(configDir string) error {
	configPath := filepath.Join(configDir, "config.yaml")

	defaultConfig := `# Code Vault Configuration
seaweedfs:
  s3_endpoint: "http://localhost:8333"
  access_key: ""
  secret_key: ""
  bucket_name: "code-vault"
  use_ssl: false

database:
  host: "localhost"
  port: 5432
  user: "vault_user"
  password: ""
  dbname: "code_vault"
  sslmode: "disable"

user:
  api_key: ""
  username: ""
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "code_vault")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}
