package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"krypt/internal/client"
	"krypt/internal/crypto"
)

// pasteUsage prints help for the paste sub-command.
func pasteUsage() {
	fmt.Fprintf(os.Stderr, `Usage: krypt paste [flags] [file]

Create an encrypted paste. Content is read from stdin by default, or from
[file] if provided. Prints the full share URL (including key fragment) to stdout.

Flags:
  --server   string   Base URL of the krypt server (default: %s)
                      Priority: --server flag > KRYPT_SERVER env > config file.
                      Run "krypt config set server <url>" to persist a default.
  --title    string   Optional encrypted title for the paste.
  --lang     string   Optional language hint (e.g. go, python, javascript).
  --password string   Optional password for double-layer encryption.
  --expires  string   Expiry duration, e.g. 10m, 1h, 7d, 30d (default: never).
  --burn              Delete paste after the first read.

Examples:
  echo "Hello, World!" | krypt paste
  echo "secret" | krypt paste --password hunter2 --expires 1h
  krypt paste --title "My Script" --lang bash script.sh
  cat /etc/hosts | krypt paste --server http://localhost:3000

`, serverDefault())
}

// RunPaste implements the "paste" sub-command.
func RunPaste(args []string) error {
	fs := flag.NewFlagSet("paste", flag.ContinueOnError)
	fs.Usage = pasteUsage

	server := fs.String("server", serverDefault(), "krypt server base URL")
	title := fs.String("title", "", "Encrypted paste title")
	lang := fs.String("lang", "", "Language hint")
	password := fs.String("password", "", "Optional password")
	expires := fs.String("expires", "", "Expiry duration (e.g. 1h, 7d)")
	burn := fs.Bool("burn", false, "Burn after read")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Read content from file argument or stdin
	var content string
	switch {
	case fs.NArg() > 0:
		data, err := os.ReadFile(fs.Arg(0))
		if err != nil {
			return fmt.Errorf("reading file %q: %w", fs.Arg(0), err)
		}
		content = string(data)
	default:
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		content = string(data)
	}

	content = strings.TrimRight(content, "\n")
	if content == "" {
		return fmt.Errorf("content is empty – nothing to paste")
	}

	// Parse expiry
	var expiresIn *int
	if *expires != "" {
		secs, err := parseDuration(*expires)
		if err != nil {
			return err
		}
		expiresIn = &secs
	}

	// Encrypt locally – key never leaves the process except in the URL fragment
	result, err := crypto.Encrypt(content, *title, *lang, *password)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Upload ciphertext to server
	c := client.New(*server)
	cr, err := c.CreatePaste(client.CreateRequest{
		EncryptedData: result.EncryptedData,
		IV:            result.IV,
		Salt:          result.Salt,
		ExpiresIn:     expiresIn,
		BurnAfterRead: *burn,
		HasPassword:   *password != "",
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	base := strings.TrimRight(*server, "/")

	// View URL: key in fragment, never in path/query
	fmt.Printf("\033[32mView:\033[0m    %s/p/%s#%s\n", base, cr.ID, result.KeyFragment)

	// Destroy URL: anyone with this link can permanently delete the paste
	fmt.Printf("\033[31mDestroy:\033[0m %s/destroy/%s?token=%s\n", base, cr.ID, cr.DeleteToken)
	fmt.Fprintf(os.Stderr, "\n\033[33mWarning:\033[0m anyone with the destroy link can permanently delete this paste\n")

	return nil
}

// parseDuration converts strings like "10m", "1h", "7d", "30d", "never" into seconds.
func parseDuration(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "never" || s == "0" {
		return 0, fmt.Errorf("use no --expires flag for a paste that never expires")
	}

	// Try standard Go duration first (handles s, m, h)
	if d, err := time.ParseDuration(s); err == nil {
		secs := int(d.Seconds())
		if secs <= 0 {
			return 0, fmt.Errorf("expires must be a positive duration")
		}
		return secs, nil
	}

	// Handle day suffix: e.g. "7d", "30d"
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid duration %q – try 10m, 1h, 7d, 30d", s)
		}
		return days * 86400, nil
	}

	// Handle week suffix: "2w"
	if strings.HasSuffix(s, "w") {
		var weeks int
		if _, err := fmt.Sscanf(s, "%dw", &weeks); err != nil || weeks <= 0 {
			return 0, fmt.Errorf("invalid duration %q – try 1w, 2w", s)
		}
		return weeks * 7 * 86400, nil
	}

	return 0, fmt.Errorf("invalid duration %q – supported: 10m, 1h, 7d, 30d, 2w", s)
}
