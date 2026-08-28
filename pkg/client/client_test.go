package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/pkg/server"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
)

type mockResolver struct {
	entries map[string]sourcetype.SourceData
	err     error
}

func (m *mockResolver) Search(_ context.Context, _ string, _ string, _ []*model.InFacet, _ []*model.InFilter, _ []float64, _ *int, _ *int, _ *string, _ []*model.SortField) (*model.SearchResult, error) {
	return nil, m.err
}

func (m *mockResolver) MediathekEntries(_ context.Context, _ []string) ([]*model.MediathekFullEntry, error) {
	return nil, m.err
}

func (m *mockResolver) ReferencesFull(_ context.Context, _ *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	return nil, m.err
}

func (m *mockResolver) LoadEntries(_ context.Context, signatures []string) ([]sourcetype.SourceData, error) {
	if m.err != nil {
		return nil, m.err
	}
	var res []sourcetype.SourceData
	for _, sig := range signatures {
		if entry, ok := m.entries[sig]; ok {
			res = append(res, entry)
		}
	}
	return res, nil
}

func (m *mockResolver) StoreEntry(_ context.Context, signature string, data *sourcetype.SourceData) error {
	if m.err != nil {
		return m.err
	}
	if m.entries == nil {
		m.entries = make(map[string]sourcetype.SourceData)
	}
	m.entries[signature] = *data
	return nil
}

func (m *mockResolver) DeleteEntry(_ context.Context, signature string) error {
	if m.err != nil {
		return m.err
	}
	if m.entries != nil {
		delete(m.entries, signature)
	}
	return nil
}

func newTestLogger() zLogger.ZLogger {
	nopLogger := zerolog.Nop()
	return &nopLogger
}

func createTestJWT(key string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(key))
}

func TestNewClient(t *testing.T) {
	t.Run("empty baseURL", func(t *testing.T) {
		_, err := New("")
		if err == nil {
			t.Fatal("expected error on empty baseURL, got nil")
		}
	})

	t.Run("valid baseURL with scheme", func(t *testing.T) {
		cli, err := New("http://localhost:8080/rest")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cli == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("valid baseURL without scheme", func(t *testing.T) {
		cli, err := NewClient("localhost:8080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cli == nil {
			t.Fatal("expected non-nil client")
		}
	})
}

func TestClientOptions(t *testing.T) {
	customHeaderKey := "X-Custom-Header"
	customHeaderVal := "custom-value"
	userAgent := "CustomAgent/1.0"
	jwtSecret := "test-secret-key"

	validToken, err := createTestJWT(jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create jwt token: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent %q, got %q", userAgent, r.Header.Get("User-Agent"))
		}
		if r.Header.Get(customHeaderKey) != customHeaderVal {
			t.Errorf("expected %s %q, got %q", customHeaderKey, customHeaderVal, r.Header.Get(customHeaderKey))
		}
		if r.Header.Get("Authorization") != "Bearer "+validToken {
			t.Errorf("expected Authorization header with valid JWT token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcetype.SourceData{Signature: "test-1"})
	}))
	defer mockServer.Close()

	customHTTPClient := &http.Client{Timeout: 10 * time.Second}

	cli, err := New(mockServer.URL,
		WithHTTPClient(customHTTPClient),
		WithUserAgent(userAgent),
		WithHeader(customHeaderKey, customHeaderVal),
		WithBearerToken(validToken),
		WithTimeout(5*time.Second),
		WithLogger(newTestLogger()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	data, err := cli.GetItem(context.Background(), "test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Signature != "test-1" {
		t.Errorf("expected signature test-1, got %s", data.Signature)
	}
}

func TestClientTokenProvider(t *testing.T) {
	providerCalled := false
	jwtSecret := "test-provider-secret"
	validToken, err := createTestJWT(jwtSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create jwt token: %v", err)
	}

	tokenProvider := func() (string, error) {
		providerCalled = true
		return validToken, nil
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+validToken {
			t.Errorf("expected Authorization header with valid provider JWT token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcetype.SourceData{Signature: "test-1"})
	}))
	defer mockServer.Close()

	cli, err := New(mockServer.URL, WithTokenProvider(tokenProvider))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = cli.GetItem(context.Background(), "test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !providerCalled {
		t.Error("expected tokenProvider to be called")
	}
}

func TestClientWithServerIntegration(t *testing.T) {
	logger := newTestLogger()
	jwtSecret := "test-sync-jwt-key"
	mockRes := &mockResolver{
		entries: map[string]sourcetype.SourceData{
			"sig-1": {
				Signature: "sig-1",
				Source:    "ub-source",
			},
		},
	}

	ctrl := server.NewController("localhost:0", "http://localhost:0/graphql", nil, mockRes, nil, jwtSecret, logger)
	ts := httptest.NewServer(ctrl.Handler())
	defer ts.Close()

	t.Run("invalid params", func(t *testing.T) {
		cli, err := New(ts.URL, WithJWTKey(jwtSecret))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "")
		if !errors.Is(err, ErrEmptySignature) {
			t.Errorf("expected ErrEmptySignature, got %v", err)
		}

		err = cli.UpdateItem(context.Background(), "", &sourcetype.SourceData{})
		if !errors.Is(err, ErrEmptySignature) {
			t.Errorf("expected ErrEmptySignature, got %v", err)
		}

		err = cli.UpdateItem(context.Background(), "sig-1", nil)
		if !errors.Is(err, ErrNilData) {
			t.Errorf("expected ErrNilData, got %v", err)
		}

		err = cli.DeleteItem(context.Background(), "")
		if !errors.Is(err, ErrEmptySignature) {
			t.Errorf("expected ErrEmptySignature, got %v", err)
		}
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		cli, err := New(ts.URL) // no auth token or key
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected HTTPError with 401, got %v", err)
		}

		err = cli.DeleteItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("unauthorized with wrong jwt secret", func(t *testing.T) {
		wrongToken, err := createTestJWT("wrong-secret-key", 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to create jwt token: %v", err)
		}

		cli, err := New(ts.URL, WithBearerToken(wrongToken))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("unauthorized with expired token", func(t *testing.T) {
		expiredToken, err := createTestJWT(jwtSecret, -1*time.Hour)
		if err != nil {
			t.Fatalf("failed to create expired jwt token: %v", err)
		}

		cli, err := New(ts.URL, WithBearerToken(expiredToken))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("get signature success with WithJWTKey", func(t *testing.T) {
		cli, err := New(ts.URL, WithJWTKey(jwtSecret), WithJWTDuration(30*time.Minute))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		data, err := cli.GetItem(context.Background(), "sig-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data.Signature != "sig-1" || data.Source != "ub-source" {
			t.Errorf("unexpected data returned: %+v", data)
		}

		// Test alias GetSignature
		dataAlias, err := cli.GetSignature(context.Background(), "sig-1")
		if err != nil {
			t.Fatalf("unexpected error with GetSignature alias: %v", err)
		}
		if dataAlias.Signature != "sig-1" {
			t.Errorf("unexpected data from GetSignature: %+v", dataAlias)
		}
	})

	t.Run("get signature success with WithBearerToken", func(t *testing.T) {
		validToken, err := createTestJWT(jwtSecret, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to create valid jwt token: %v", err)
		}

		cli, err := New(ts.URL, WithBearerToken(validToken))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		data, err := cli.GetItem(context.Background(), "sig-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data.Signature != "sig-1" || data.Source != "ub-source" {
			t.Errorf("unexpected data returned: %+v", data)
		}
	})

	t.Run("get signature success with WithTokenProvider", func(t *testing.T) {
		cli, err := New(ts.URL, WithTokenProvider(func() (string, error) {
			return createTestJWT(jwtSecret, 1*time.Hour)
		}))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		data, err := cli.GetItem(context.Background(), "sig-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data.Signature != "sig-1" || data.Source != "ub-source" {
			t.Errorf("unexpected data returned: %+v", data)
		}
	})

	t.Run("get signature not found", func(t *testing.T) {
		cli, err := New(ts.URL, WithJWTKey(jwtSecret))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "non-existing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("update signature success", func(t *testing.T) {
		cli, err := New(ts.URL, WithJWTKey(jwtSecret))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		newData := &sourcetype.SourceData{
			Signature: "sig-2",
			Source:    "new-source",
		}

		err = cli.UpdateItem(context.Background(), "sig-2", newData)
		if err != nil {
			t.Fatalf("unexpected error updating item: %v", err)
		}

		stored, ok := mockRes.entries["sig-2"]
		if !ok || stored.Source != "new-source" {
			t.Errorf("entry was not stored correctly: %+v", stored)
		}

		// Test alias UpdateSignature
		newData.Source = "updated-source"
		err = cli.UpdateSignature(context.Background(), "sig-2", newData)
		if err != nil {
			t.Fatalf("unexpected error updating item via alias: %v", err)
		}
		if mockRes.entries["sig-2"].Source != "updated-source" {
			t.Errorf("entry was not updated correctly: %+v", mockRes.entries["sig-2"])
		}

		// Verify retrieval of updated entry via GetItem
		fetched, err := cli.GetItem(context.Background(), "sig-2")
		if err != nil {
			t.Fatalf("unexpected error fetching updated item: %v", err)
		}
		if fetched.Signature != "sig-2" || fetched.Source != "updated-source" {
			t.Errorf("unexpected data retrieved for sig-2: %+v", fetched)
		}
	})

	t.Run("delete signature success", func(t *testing.T) {
		cli, err := New(ts.URL, WithJWTKey(jwtSecret))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Store an item first
		data := &sourcetype.SourceData{Signature: "sig-delete-1", Source: "delete-source"}
		if err := cli.UpdateItem(context.Background(), "sig-delete-1", data); err != nil {
			t.Fatalf("failed to setup item for deletion: %v", err)
		}

		// Delete item
		if err := cli.DeleteItem(context.Background(), "sig-delete-1"); err != nil {
			t.Fatalf("unexpected error deleting item: %v", err)
		}

		if _, ok := mockRes.entries["sig-delete-1"]; ok {
			t.Errorf("entry was not deleted from mock resolver")
		}

		// Test delete alias DeleteSignature
		dataAlias := &sourcetype.SourceData{Signature: "sig-delete-2", Source: "delete-source-2"}
		if err := cli.UpdateItem(context.Background(), "sig-delete-2", dataAlias); err != nil {
			t.Fatalf("failed to setup item for alias deletion: %v", err)
		}

		if err := cli.DeleteSignature(context.Background(), "sig-delete-2"); err != nil {
			t.Fatalf("unexpected error deleting item via alias: %v", err)
		}

		if _, ok := mockRes.entries["sig-delete-2"]; ok {
			t.Errorf("entry was not deleted from mock resolver via alias")
		}
	})

	t.Run("server internal error", func(t *testing.T) {
		errResolver := &mockResolver{err: errors.New("database failure")}
		errCtrl := server.NewController("localhost:0", "http://localhost:0/graphql", nil, errResolver, nil, jwtSecret, logger)
		errTs := httptest.NewServer(errCtrl.Handler())
		defer errTs.Close()

		cli, err := New(errTs.URL, WithJWTKey(jwtSecret))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = cli.GetItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error on server 500, got nil")
		}
		if !errors.Is(err, ErrInternalServerError) {
			t.Errorf("expected ErrInternalServerError, got %v", err)
		}

		err = cli.UpdateItem(context.Background(), "sig-1", &sourcetype.SourceData{Signature: "sig-1"})
		if err == nil {
			t.Fatal("expected error on server 500, got nil")
		}
		if !errors.Is(err, ErrInternalServerError) {
			t.Errorf("expected ErrInternalServerError, got %v", err)
		}

		err = cli.DeleteItem(context.Background(), "sig-1")
		if err == nil {
			t.Fatal("expected error on server 500, got nil")
		}
		if !errors.Is(err, ErrInternalServerError) {
			t.Errorf("expected ErrInternalServerError, got %v", err)
		}
	})
}

func TestHTTPErrorHandling(t *testing.T) {
	httpErr := &HTTPError{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Message:    "invalid json body",
	}

	if !errors.Is(httpErr, ErrBadRequest) {
		t.Errorf("expected errors.Is(httpErr, ErrBadRequest) to be true")
	}
	if errors.Is(httpErr, ErrNotFound) {
		t.Errorf("expected errors.Is(httpErr, ErrNotFound) to be false")
	}
	expectedStr := "http error 400 (400 Bad Request): invalid json body"
	if httpErr.Error() != expectedStr {
		t.Errorf("expected %q, got %q", expectedStr, httpErr.Error())
	}
}
