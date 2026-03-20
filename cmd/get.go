package cmd

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"krypt/internal/client"
	"krypt/internal/crypto"
)

// getUsage prints help for the get sub-command.
func getUsage() {
	fmt.Fprintf(os.Stderr, `Usage: krypt get [flags] <url>

Retrieve and decrypt a paste. The URL must include the key fragment (#...).
Prints decrypted content to stdout.

Flags:
  --password string   Password, if the paste is password-protected.
  --meta              Also print title and language metadata to stderr.

Examples:
  krypt get 'https://paste.example.com/p/abc123#mysecretkey'
  krypt get --password hunter2 'https://paste.example.com/p/xyz#key'

`)
}

// RunGet implements the "get" sub-command.
func RunGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.Usage = getUsage

	password := fs.String("password", "", "Password for password-protected pastes")
	showMeta := fs.Bool("meta", false, "Print title/language metadata to stderr")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		getUsage()
		return fmt.Errorf("expected exactly one URL argument")
	}

	rawURL := fs.Arg(0)
	pasteID, keyFragment, serverBase, err := parseShareURL(rawURL)
	if err != nil {
		return err
	}

	if keyFragment == "" {
		return fmt.Errorf("URL is missing the key fragment (#...) – decryption is impossible without it")
	}

	// Fetch encrypted data from server
	c := client.New(serverBase)
	pd, err := c.GetPaste(pasteID)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	// Check if password is needed but not provided
	if pd.HasPassword && *password == "" {
		return fmt.Errorf("this paste is password-protected – supply --password <pass>")
	}

	// Decrypt locally
	payload, err := crypto.Decrypt(pd.EncryptedData, pd.IV, pd.Salt, keyFragment, *password)
	if err != nil {
		return err
	}

	// Metadata summary (to stderr so it doesn't pollute piped output)
	if *showMeta {
		if payload.Title != "" {
			fmt.Fprintf(os.Stderr, "Title:    %s\n", payload.Title)
		}
		if payload.Language != "" {
			fmt.Fprintf(os.Stderr, "Language: %s\n", payload.Language)
		}
		fmt.Fprintf(os.Stderr, "Reads:    %d\n", pd.ReadCount)
		if pd.BurnAfterRead {
			fmt.Fprintln(os.Stderr, "Note:     this was a burn-after-read paste (now deleted)")
		}
		if pd.ExpiresAt != nil {
			exp := time.Unix(*pd.ExpiresAt, 0)
			remaining := time.Until(exp).Round(time.Second)
			if remaining > 0 {
				fmt.Fprintf(os.Stderr, "Expires:  %s (in %s)\n", exp.Format(time.RFC3339), remaining)
			} else {
				fmt.Fprintf(os.Stderr, "Expires:  %s (expired)\n", exp.Format(time.RFC3339))
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Print(payload.Content)
	// Ensure output ends with a newline when writing to a terminal
	if !strings.HasSuffix(payload.Content, "\n") {
		fmt.Println()
	}
	return nil
}

// parseShareURL breaks a krypt share URL into its components.
// Expected form: {scheme}://{host}/p/{id}#{keyFragment}
func parseShareURL(rawURL string) (id, keyFragment, serverBase string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", "", fmt.Errorf("URL must start with http:// or https://")
	}

	// Path must be /p/{id}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "p" || parts[1] == "" {
		return "", "", "", fmt.Errorf("URL path must be /p/<id>, got %q", u.Path)
	}

	id = parts[1]
	keyFragment = u.Fragment
	serverBase = u.Scheme + "://" + u.Host

	return id, keyFragment, serverBase, nil
}
