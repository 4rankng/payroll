// argoncheck verifies candidate passwords against an encoded Argon2id hash
// using the exact production verification path (peppering included), so an
// offline check can never disagree with what login would accept.
//
// Usage:
//
//	HASH_SECRET=... HASH_SALT=... go run ./cmd/argoncheck <hash-file> [candidate ...]
//
// Hash file holds one encoded hash ($argon2id$v=..$m=..$salt$hash). Without
// candidate arguments, one candidate is read per line from stdin. Secrets
// come from the same HASH_SECRET / HASH_SALT env vars the server uses; an
// empty pepper (unset vars) matches a hash that was created without one.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"api-server/internal/app/services/user"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: argoncheck <hash-file> [candidate ...]  (candidates on stdin when omitted)")
		os.Exit(2)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read hash file:", err)
		os.Exit(2)
	}
	encodedHash := strings.TrimSpace(string(data))
	if encodedHash == "" {
		fmt.Fprintln(os.Stderr, "hash file is empty")
		os.Exit(2)
	}

	candidates := os.Args[2:]
	if len(candidates) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); line != "" {
				candidates = append(candidates, line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "read stdin:", err)
			os.Exit(2)
		}
	}
	if len(candidates) == 0 {
		fmt.Fprintln(os.Stderr, "no candidates given")
		os.Exit(2)
	}

	hashSecret := os.Getenv("HASH_SECRET")
	hashSalt := os.Getenv("HASH_SALT")

	matched := false
	for _, candidate := range candidates {
		if user.VerifyArgon2idHash(candidate, encodedHash, hashSecret, hashSalt) {
			fmt.Println("MATCH:", candidate)
			matched = true
		} else {
			fmt.Println("no:  ", candidate)
		}
	}
	if !matched {
		os.Exit(1)
	}
}
