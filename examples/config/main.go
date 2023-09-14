package main

import (
	"os"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/vault"
	"github.com/knadh/koanf/v2"
	"github.com/struqt/logging"
)

func main() {
	var config = koanf.New(".")
	logging.LogConsoleThreshold = 0
	log := logging.NewLogger("").WithName("Config")
	f1 := file.Provider("mock/config.yml")
	if err := config.Load(f1, yaml.Parser()); err != nil {
		log.Error(err, "Failed to load config")
		return
	}
	f2 := file.Provider("mock/config.toml")
	if err := config.Load(f2, toml.Parser()); err != nil {
		log.Error(err, "Failed to load config")
		return
	}
	if err := loadFromVault(config); err != nil {
		log.Error(err, "Failed to load config")
		return
	}
	log.Info("",
		"vault.address", config.String("vault.address"),
		"id", config.Int("parent1.id"),
		"type", config.String("parent1.child1.type"),
		"name", config.String("parent1.name"),
		"raw", config.Raw(),
		"V_TEST_001", config.String("data.TEST_001"),
		"V_TEST_002", config.String("TEST_002"),
	)
}

type VaultPathConfig struct {
	Token   string `koanf:"token"`
	Address string `koanf:"address"`
	Path    string `koanf:"path"`
}

func loadFromVault(k *koanf.Koanf) error {
	var raw VaultPathConfig
	if err := k.Unmarshal("vault", &raw); err != nil {
		return err
	}
	config := vault.Config{
		Address:     replaceWithEnv(raw.Address, "https://vault.example.com"),
		Token:       replaceWithEnv(raw.Token, "s.abc...***"),
		Path:        replaceWithEnv(raw.Path, "storage/data/project/demo"),
		FlatPaths:   true,
		Delim:       ".",
		Timeout:     15 * time.Second,
		ExcludeMeta: true,
	}
	return k.Load(vault.Provider(config), nil)
}

func replaceWithEnv(val string, init string) string {
	if len(val) > 1 && val[0] == '$' {
		s := os.Getenv(val[1:])
		if len(s) > 0 {
			return s
		}
	}
	if len(val) > 0 {
		return val
	}
	if len(init) > 0 {
		return init
	}
	return val
}
