// Package cmd holds all CLI sub-commands for krypt.
package cmd

import "krypt/internal/config"

// serverDefault returns the effective server URL using the priority chain:
//  1. KRYPT_SERVER environment variable
//  2. server key in ~/.config/krypt/config.toml
//  3. built-in fallback (http://localhost:3000)
//
// The --server flag sits above all of these and is applied in each sub-command.
func serverDefault() string {
	return config.ServerDefault()
}
