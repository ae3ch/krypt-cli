// krypt – CLI tool for the krypt encrypted pastebin.
//
// Usage:
//
//	echo "secret" | krypt paste [flags]           # create paste, print share URL
//	krypt get [flags] <url>                        # fetch + decrypt paste
//	krypt config set server <url>                  # set default server
//
// The encryption key is embedded in the URL fragment (#...) and is NEVER
// sent to the server.  All cryptography runs locally.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"krypt/cmd"
	"krypt/internal/config"
)

const usage = `krypt — zero-knowledge encrypted pastebin CLI

Usage:
  krypt paste  [flags] [file]          Create an encrypted paste
  krypt get    [flags] <url>           Retrieve and decrypt a paste
  krypt config set server <url>        Set default server URL
  krypt config get server              Print current server URL
  krypt config path                    Print config file location

Run "krypt <command> --help" for per-command flags.

Environment:
  KRYPT_SERVER   Override the default server URL for this session.

Priority (highest to lowest):
  --server flag  >  KRYPT_SERVER env  >  config file  >  built-in default
`

// promptServer asks the user for their server URL on first run and saves it.
func promptServer() {
	fmt.Fprintf(os.Stderr, "Welcome to krypt! Please enter your server URL.\n")
	fmt.Fprintf(os.Stderr, "Server URL (e.g. https://krypt.li): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	url := strings.TrimSpace(scanner.Text())
	if url == "" {
		fmt.Fprintf(os.Stderr, "No server set. You can set one later with: krypt config set server <url>\n\n")
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if err := config.Save(&config.Config{Server: url}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save server URL: %v\n\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Server set to %s\n\n", url)
}

func main() {
	// Ensure the config file exists on every invocation (first-run creation).
	if created, err := config.EnsureCreated(); err != nil {
		// Non-fatal: print a warning but continue.
		fmt.Fprintf(os.Stderr, "warning: could not create config file: %v\n", err)
	} else if created {
		p, _ := config.Path()
		fmt.Fprintf(os.Stderr, "Created default config file: %s\n\n", p)
		promptServer()
	}

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "paste", "p":
		err = cmd.RunPaste(os.Args[2:])
	case "get", "g":
		err = cmd.RunGet(os.Args[2:])
	case "config", "cfg":
		err = cmd.RunConfig(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	case "version", "--version", "-v":
		fmt.Println("krypt version 0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
