package handler

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/configwizard"
)

type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func (h *ConfigHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage Pumbaa configuration",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Interactive configuration wizard",
				Action: func(c *cli.Context) error {
					return configwizard.ConfigWizard()
				},
			},
			{
				Name:      "set",
				Usage:     "Set a configuration value",
				ArgsUsage: "<key> <value>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						return fmt.Errorf("usage: pumbaa config set <key> <value>")
					}
					return h.Set(c.Args().Get(0), c.Args().Get(1))
				},
			},
			{
				Name:      "get",
				Usage:     "Get a configuration value",
				ArgsUsage: "<key>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						return fmt.Errorf("usage: pumbaa config get <key>")
					}
					return h.Get(c.Args().Get(0))
				},
			},
			{
				Name:  "list",
				Usage: "List all configuration values",
				Action: func(c *cli.Context) error {
					return h.List()
				},
			},
			{
				Name:  "path",
				Usage: "Show configuration file path",
				Action: func(c *cli.Context) error {
					fmt.Println(config.DefaultConfigPath())
					return nil
				},
			},
		},
	}
}

func (h *ConfigHandler) Set(key, value string) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetValue(key, value); err != nil {
		return err
	}

	if err := config.SaveFileConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}

func (h *ConfigHandler) Get(key string) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}

	value, found := cfg.GetValue(key)
	if !found {
		fmt.Printf("%s: (not set)\n", key)
	} else {
		// Mask API keys for security
		if strings.Contains(key, "api_key") && len(value) > 8 {
			value = value[:4] + "..." + value[len(value)-4:]
		}
		fmt.Printf("%s: %s\n", key, value)
	}
	return nil
}

func (h *ConfigHandler) List() error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Configuration file: %s\n\n", config.DefaultConfigPath())

	for _, key := range config.AllKeys() {
		value, found := cfg.GetValue(key)
		if found {
			// Mask API keys for security
			if strings.Contains(key, "api_key") && len(value) > 8 {
				value = value[:4] + "..." + value[len(value)-4:]
			}
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	return nil
}
