package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"krypt/internal/config"
)

func configUsage() {
	fmt.Fprintf(os.Stderr, `Usage: krypt config <subcommand> [args]

Manage the krypt configuration file (%s).

Subcommands:
  set server <url>   Set the default server URL
  get server         Print the current server URL
  path               Print the path to the config file

Examples:
  krypt config set server https://paste.example.com
  krypt config get server
  krypt config path

`, configPath())
}

// RunConfig implements the "config" sub-command.
func RunConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.Usage = configUsage
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		configUsage()
		return fmt.Errorf("expected a subcommand (set, get, path)")
	}

	switch fs.Arg(0) {
	case "set":
		return runConfigSet(fs.Args()[1:])
	case "get":
		return runConfigGet(fs.Args()[1:])
	case "path":
		fmt.Println(configPath())
		return nil
	default:
		return fmt.Errorf("unknown config subcommand %q – use set, get, or path", fs.Arg(0))
	}
}

// runConfigSet handles: krypt config set <key> <value>
func runConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: krypt config set <key> <value>\n  e.g. krypt config set server https://paste.example.com")
	}
	key := strings.ToLower(args[0])
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "server":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("server URL must start with http:// or https://")
		}
		cfg.Server = strings.TrimRight(value, "/")
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Config saved: server = %q\n", cfg.Server)
		fmt.Fprintf(os.Stderr, "  (%s)\n", configPath())
	default:
		return fmt.Errorf("unknown config key %q – supported keys: server", key)
	}
	return nil
}

// runConfigGet handles: krypt config get <key>
func runConfigGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: krypt config get <key>\n  e.g. krypt config get server")
	}
	key := strings.ToLower(args[0])

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "server":
		effective := config.ServerDefault()
		if cfg.Server != "" {
			fmt.Printf("%s\n", cfg.Server)
		} else {
			fmt.Printf("%s  (default, not set in config)\n", effective)
		}
	default:
		return fmt.Errorf("unknown config key %q – supported keys: server", key)
	}
	return nil
}

// configPath returns the config file path for display, falling back to "?" on error.
func configPath() string {
	p, err := config.Path()
	if err != nil {
		return "?"
	}
	return p
}
