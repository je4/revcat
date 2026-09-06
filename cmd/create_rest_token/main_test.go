package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		wantKey     string
		wantDur     time.Duration
		wantSub     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "flag -jwtkey",
			args:    []string{"-jwtkey", "secret123"},
			wantKey: "secret123",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "flag -key",
			args:    []string{"-key", "secret-alias"},
			wantKey: "secret-alias",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "flag -k",
			args:    []string{"-k", "secret-k"},
			wantKey: "secret-k",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "positional argument",
			args:    []string{"pos-secret"},
			wantKey: "pos-secret",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "env var REVCAT_JWT_KEY",
			args:    []string{},
			env:     map[string]string{"REVCAT_JWT_KEY": "env-secret"},
			wantKey: "env-secret",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "env var JWT_KEY",
			args:    []string{},
			env:     map[string]string{"JWT_KEY": "env-jwt-key"},
			wantKey: "env-jwt-key",
			wantDur: 1 * time.Hour,
		},
		{
			name:    "custom duration and subject",
			args:    []string{"-jwtkey", "secret", "-duration", "3h", "-subject", "test-user"},
			wantKey: "secret",
			wantDur: 3 * time.Hour,
			wantSub: "test-user",
		},
		{
			name:    "custom duration alias -d and -sub",
			args:    []string{"-jwtkey", "secret", "-d", "2h", "-sub", "test-sub"},
			wantKey: "secret",
			wantDur: 2 * time.Hour,
			wantSub: "test-sub",
		},
		{
			name:        "missing key error",
			args:        []string{},
			wantErr:     true,
			errContains: "missing required JWT key",
		},
		{
			name:        "duration <= 0 error",
			args:        []string{"-jwtkey", "secret", "-duration", "0s"},
			wantErr:     true,
			errContains: "duration must be greater than 0",
		},
		{
			name:        "duration > 4h error",
			args:        []string{"-jwtkey", "secret", "-duration", "5h"},
			wantErr:     true,
			errContains: "duration must not exceed 4h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				if tt.env != nil {
					return tt.env[key]
				}
				return ""
			}

			cfg, err := parseFlags(tt.args, getenv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if cfg.JWTKey != tt.wantKey {
				t.Errorf("cfg.JWTKey = %q, want %q", cfg.JWTKey, tt.wantKey)
			}
			if cfg.Duration != tt.wantDur {
				t.Errorf("cfg.Duration = %v, want %v", cfg.Duration, tt.wantDur)
			}
			if cfg.Subject != tt.wantSub {
				t.Errorf("cfg.Subject = %q, want %q", cfg.Subject, tt.wantSub)
			}
		})
	}
}

func TestGenerateTokenAndValidate(t *testing.T) {
	key := "my-very-secret-jwt-key"
	dur := 2 * time.Hour
	sub := "operator-1"

	cfg := &Config{
		JWTKey:   key,
		Duration: dur,
		Subject:  sub,
	}

	tokenStr, err := GenerateToken(cfg)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("GenerateToken() returned empty token string")
	}

	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuedAt())

	if err != nil {
		t.Fatalf("jwt.ParseWithClaims failed: %v", err)
	}
	if !parsedToken.Valid {
		t.Fatal("parsed token is not valid")
	}

	if claims.Subject != sub {
		t.Errorf("claims.Subject = %q, want %q", claims.Subject, sub)
	}

	now := time.Now()
	if claims.IssuedAt == nil || claims.IssuedAt.After(now.Add(1*time.Minute)) || claims.IssuedAt.Before(now.Add(-1*time.Minute)) {
		t.Errorf("claims.IssuedAt (%v) is not close to current time (%v)", claims.IssuedAt, now)
	}

	expectedExp := claims.IssuedAt.Time.Add(dur)
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(expectedExp) {
		t.Errorf("claims.ExpiresAt = %v, want %v", claims.ExpiresAt, expectedExp)
	}

	// Verify maximum lifetime rule as enforced by validateJWT in pkg/server
	maxAge := 4 * time.Hour
	if claims.IssuedAt.Time.Add(maxAge).Before(claims.ExpiresAt.Time) {
		t.Errorf("token lifetime (%v) exceeds maxAge (%v)", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time), maxAge)
	}
}

func TestRun(t *testing.T) {
	var stdout, stderr bytes.Buffer
	key := "integration-test-secret"

	exitCode := run([]string{"-jwtkey", key, "-duration", "30m"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exitCode 0, got %d, stderr: %s", exitCode, stderr.String())
	}

	tokenStr := strings.TrimSpace(stdout.String())
	if tokenStr == "" {
		t.Fatal("expected non-empty token string in stdout")
	}

	// Missing key should return non-zero exit code
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("expected exitCode 1 on missing key, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "missing required JWT key") {
		t.Errorf("expected stderr to contain missing key error, got: %s", stderr.String())
	}
}
