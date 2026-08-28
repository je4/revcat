package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/config"
	uconfig "github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
)

func newTestLogger() zLogger.ZLogger {
	l := zerolog.Nop()
	return &l
}

func generateTestToken(secret string, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestExtractBearerToken(t *testing.T) {
	logger := newTestLogger()

	tests := []struct {
		name       string
		authHeader string
		wantToken  string
		wantErr    bool
	}{
		{
			name:       "empty header",
			authHeader: "",
			wantToken:  "",
			wantErr:    true,
		},
		{
			name:       "wrong prefix",
			authHeader: "Basic 123456",
			wantToken:  "",
			wantErr:    true,
		},
		{
			name:       "valid bearer token",
			authHeader: "Bearer my-secret-token",
			wantToken:  "my-secret-token",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractBearerToken(tt.authHeader, logger)
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractBearerToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if token != tt.wantToken {
				t.Errorf("extractBearerToken() = %q, want %q", token, tt.wantToken)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	logger := newTestLogger()
	secret := "test-secret-key"

	now := time.Now()
	validClaims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
	}
	validToken, err := generateTestToken(secret, validClaims)
	if err != nil {
		t.Fatalf("failed to generate valid token: %v", err)
	}

	expiredClaims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
	}
	expiredToken, err := generateTestToken(secret, expiredClaims)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	excessiveLifetimeClaims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Hour)),
	}
	excessiveToken, err := generateTestToken(secret, excessiveLifetimeClaims)
	if err != nil {
		t.Fatalf("failed to generate excessive lifetime token: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		key         string
		maxAge      time.Duration
		wantErr     bool
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			key:         secret,
			maxAge:      2 * time.Hour,
			wantErr:     false,
		},
		{
			name:        "invalid secret key",
			tokenString: validToken,
			key:         "wrong-secret",
			maxAge:      2 * time.Hour,
			wantErr:     true,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			key:         secret,
			maxAge:      2 * time.Hour,
			wantErr:     true,
		},
		{
			name:        "lifetime exceeds max age",
			tokenString: excessiveToken,
			key:         secret,
			maxAge:      4 * time.Hour,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateJWT(tt.tokenString, tt.key, &jwt.RegisteredClaims{}, tt.maxAge, logger)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateJWT() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := newTestLogger()
	syncKey := "sync-secret"

	r := gin.New()
	r.Use(restAuthMiddleware(syncKey, logger))
	r.GET("/test", func(c *gin.Context) {
		errVal := c.Request.Context().Value("error")
		c.JSON(http.StatusOK, gin.H{"error": errVal})
	})

	now := time.Now()
	validToken, err := generateTestToken(syncKey, &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("failed to generate valid token: %v", err)
	}

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid header format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Basic invalid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid token returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.payload")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("valid token passes without context error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestGraphqlAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := newTestLogger()

	clientApiKey := "my-api-key"
	clientSecret := "client-jwt-secret"
	client := &config.Client{
		Name:      "test-client",
		Apikey:    uconfig.EnvString(clientApiKey),
		JWTKey:    uconfig.EnvString(clientSecret),
		JWTMaxAge: uconfig.Duration(2 * time.Hour),
		Groups:    []string{"group1", "group2"},
	}
	clientMap := map[string]*config.Client{
		clientApiKey: client,
	}

	r := gin.New()
	r.Use(graphqlAuthMiddleware(clientMap, logger))
	r.GET("/graphql", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"error":  c.Request.Context().Value("error"),
			"client": c.Request.Context().Value("client"),
			"groups": c.Request.Context().Value("groups"),
		})
	})

	now := time.Now()
	jwtToken, err := generateTestToken(clientSecret, &groupClaims{
		Groups: "custom1;custom2",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
	})
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		expectedBody := `{"client":null,"error":"no authorization header","groups":null}`
		if w.Body.String() != expectedBody {
			t.Errorf("expected %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("invalid api key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		req.Header.Set("Authorization", "Bearer wrong-key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		expectedBody := `{"client":null,"error":"invalid application key 'wrong-key'","groups":null}`
		if w.Body.String() != expectedBody {
			t.Errorf("expected %s, got %s", expectedBody, w.Body.String())
		}
	})

	t.Run("api key only sets default client groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		req.Header.Set("Authorization", "Bearer "+clientApiKey)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("api key + valid JWT sets token groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s.%s", clientApiKey, jwtToken))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestNewController(t *testing.T) {
	logger := newTestLogger()
	ctrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, nil, nil, "secret", logger)
	if ctrl == nil {
		t.Fatal("expected non-nil Controller")
	}
	if ctrl.srv == nil {
		t.Fatal("expected non-nil server")
	}
}
