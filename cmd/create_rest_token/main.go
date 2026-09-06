package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const maxDuration = 4 * time.Hour

type Config struct {
	JWTKey   string
	Duration time.Duration
	Subject  string
}

func parseFlags(args []string, getenv func(string) string) (*Config, error) {
	fs := flag.NewFlagSet("create_rest_token", flag.ContinueOnError)

	var jwtKey string
	var keyAlias string
	var duration time.Duration
	var durationAlias time.Duration
	var subject string
	var subAlias string

	fs.StringVar(&jwtKey, "jwtkey", "", "JWT secret key for signing")
	fs.StringVar(&keyAlias, "key", "", "JWT secret key for signing (alias for -jwtkey)")
	fs.StringVar(&keyAlias, "k", "", "JWT secret key for signing (alias for -jwtkey)")

	fs.DurationVar(&duration, "duration", 1*time.Hour, "Token lifetime duration (e.g. 1h, 30m, max 4h)")
	fs.DurationVar(&durationAlias, "lifetime", 0, "Token lifetime duration (alias for -duration)")
	fs.DurationVar(&durationAlias, "d", 0, "Token lifetime duration (alias for -duration)")

	fs.StringVar(&subject, "subject", "", "Optional JWT subject claim")
	fs.StringVar(&subAlias, "sub", "", "Optional JWT subject claim (alias for -subject)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	finalKey := jwtKey
	if finalKey == "" {
		finalKey = keyAlias
	}
	if finalKey == "" && fs.NArg() > 0 {
		finalKey = fs.Arg(0)
	}
	if finalKey == "" && getenv != nil {
		if envVal := getenv("REVCAT_JWT_KEY"); envVal != "" {
			finalKey = envVal
		} else if envVal := getenv("JWT_KEY"); envVal != "" {
			finalKey = envVal
		}
	}

	if finalKey == "" {
		return nil, errors.New("missing required JWT key: specify via -jwtkey, -key, positional argument, or REVCAT_JWT_KEY env var")
	}

	finalDuration := duration
	if durationAlias > 0 {
		finalDuration = durationAlias
	}

	if finalDuration <= 0 {
		return nil, fmt.Errorf("invalid duration %v: duration must be greater than 0", finalDuration)
	}
	if finalDuration > maxDuration {
		return nil, fmt.Errorf("invalid duration %v: duration must not exceed %v", finalDuration, maxDuration)
	}

	finalSubject := subject
	if finalSubject == "" {
		finalSubject = subAlias
	}

	return &Config{
		JWTKey:   finalKey,
		Duration: finalDuration,
		Subject:  finalSubject,
	}, nil
}

func GenerateToken(cfg *Config) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.Duration)),
	}
	if cfg.Subject != "" {
		claims.Subject = cfg.Subject
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage of create_rest_token:")
	fmt.Fprintln(w, "  create_rest_token -jwtkey <key> [-duration <dur>] [-subject <sub>]")
	fmt.Fprintln(w, "  create_rest_token <key>")
	fmt.Fprintln(w, "\nFlags:")
	fmt.Fprintln(w, "  -jwtkey, -key, -k string")
	fmt.Fprintln(w, "        JWT secret key for signing")
	fmt.Fprintln(w, "  -duration, -lifetime, -d duration")
	fmt.Fprintln(w, "        Token lifetime duration (e.g. 1h, 30m, max 4h) (default 1h0m0s)")
	fmt.Fprintln(w, "  -subject, -sub string")
	fmt.Fprintln(w, "        Optional JWT subject claim")
}

func run(args []string, getenv func(string) string, stdout io.Writer, stderr io.Writer) int {
	cfg, err := parseFlags(args, getenv)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n\n", err)
		printUsage(stderr)
		return 1
	}

	tokenString, err := GenerateToken(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "Error generating token: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, tokenString)
	return 0
}

func main() {
	exitCode := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
