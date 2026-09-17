// Command hashpw produces an Argon2id encoded hash for a cleartext password
// using the same peppers and parameters the running API server uses, so the
// resulting hash can be written directly into users.password and accepted on
// login.
//
// Intended for dev use (Makefile reset-passwords / restore). Do not expose in
// production.
package main

import (
	"fmt"
	"os"

	"api-server/internal/app/services/user"
	"api-server/internal/config"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: hashpw <password>")
		os.Exit(2)
	}
	password := os.Args[1]

	cfg, err := config.LoadSecurityConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	hash, err := user.HashPassword(password, cfg.HashSecret, cfg.HashSalt, user.DefaultHashConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(hash)
}
