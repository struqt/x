package main

import (
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/struqt/x/logging"
)

var config = koanf.New(".")

func main() {
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
	log.Info("",
		"id", config.Int("parent1.id"),
		"type", config.String("parent1.child1.type"),
		"name", config.String("parent1.name"),
		"raw", config.Raw(),
	)
}
