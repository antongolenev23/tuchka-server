package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env           string           `yaml:"env" env-required:"true"`
	StoragePath   string           `yaml:"storage_path" env-default:"/var/lib/tuchka-server"`
	HTTPServerCfg HTTPServerConfig `yaml:"http_server"`
	DatabaseDSN   string           `env:"DATABASE_DSN" env-required:"true"`
}

type HTTPServerConfig struct {
	Address             string        `yaml:"address" env-required:"true"`
	RequestReadTimeout  time.Duration `yaml:"request_read_timeout" env-required:"true"`
	ResponceReadTimeout time.Duration `yaml:"responce_write_timeout" env-required:"true"`
	IdleTimeout         time.Duration `yaml:"idle_timeout" env-required:"true"`
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

	return cfg
}
