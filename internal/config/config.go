package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

type Config struct {
	Env        string           `yaml:"env" env-required:"true"`
	Database   DatabaseConfig   `yaml:"database"`
	HTTPServer HTTPServerConfig `yaml:"http_server"`
	Auth       AuthConfig
	Files      FilesConfig `yaml:"files"`
}

type DatabaseConfig struct {
	Name     string `yaml:"name" env-required:"true"`
	User     string `yaml:"user" env-required:"true"`
	Password string `env:"DB_PASSWORD" env-required:"true"`
	SSLMode  string `yaml:"sslmode" env-required:"true"`
}

type HTTPServerConfig struct {
	Address              string        `yaml:"address" env-required:"true"`
	RequestReadTimeout   time.Duration `yaml:"request_read_timeout" env-required:"true"`
	ResponseWriteTimeout time.Duration `yaml:"response_write_timeout" env-required:"true"`
	IdleTimeout          time.Duration `yaml:"idle_timeout" env-required:"true"`
}

type AuthConfig struct {
	JWTSecret          string `env:"JWT_SECRET" env-required:"true"`
	JWTExpirationHours int    `env:"JWT_EXPIRATION_HOURS" env-required:"true"`
}

type FilesConfig struct {
	StorageDir  string `yaml:"storage_dir" env-default:"/var/lib/tuchka-server"`
	MaxDownload int    `yaml:"max_download" env-default:"30"`
	MaxDelete   int    `yaml:"max_delete" env-default:"60"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	configPath, ok := os.LookupEnv("CONFIG_PATH")
	if !ok {
		log.Fatal("environment variable CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file is not exists: %s", configPath)
	}

	cfg := &Config{}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("config(%s) read error: %s", configPath, err)
	}

	err := checkStorageDir(cfg.Files.StorageDir)
	if err != nil {
		log.Fatalf("check storage path %s failed: %s", cfg.Files.StorageDir, err)
	}

	return cfg
}

func checkStorageDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("storage dir does not exist")
		}
		return errors.New("cannot access storage dir")
	}

	if !info.IsDir() {
		return errors.New("path is not directory")
	}

	tempPath := filepath.Join(path, ".temp_file")
	if err = os.WriteFile(tempPath, []byte("temp"), 0644); err != nil {
		return errors.New("storage path is not writable")
	}
	os.Remove(tempPath)

	return nil
}
