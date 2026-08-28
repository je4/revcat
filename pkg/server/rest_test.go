package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

type mockResolver struct {
	entries map[string]sourcetype.SourceData
	err     error
}

func (m *mockResolver) Search(ctx context.Context, searchType string, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string, sort []*model.SortField) (*model.SearchResult, error) {
	return nil, m.err
}

func (m *mockResolver) MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error) {
	return nil, m.err
}

func (m *mockResolver) ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	return nil, m.err
}

func (m *mockResolver) LoadEntries(ctx context.Context, signatures []string) ([]sourcetype.SourceData, error) {
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

func (m *mockResolver) StoreEntry(ctx context.Context, signature string, data *sourcetype.SourceData) error {
	if m.err != nil {
		return m.err
	}
	if m.entries == nil {
		m.entries = make(map[string]sourcetype.SourceData)
	}
	m.entries[signature] = *data
	return nil
}

func TestSwaggerEndpoints(t *testing.T) {
	logger := newTestLogger()
	ctrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, &mockResolver{}, nil, "jwt-secret", logger)

	t.Run("swagger index.html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "swagger") {
			t.Errorf("expected body to contain 'swagger', got: %s", w.Body.String())
		}
	})

	t.Run("swagger doc.json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		var doc map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
			t.Fatalf("failed to parse doc.json: %v", err)
		}
		info, ok := doc["info"].(map[string]any)
		if !ok || info["title"] != "RevCat REST API" {
			t.Errorf("expected doc.json info.title 'RevCat REST API', got %v", info)
		}
		paths, ok := doc["paths"].(map[string]any)
		if !ok || paths["/item/{signature}"] == nil {
			t.Errorf("expected doc.json paths to contain '/item/{signature}', got %v", paths)
		}
	})
}

func TestRestEndpoints(t *testing.T) {
	logger := newTestLogger()
	secret := "test-sync-key"
	mockRes := &mockResolver{
		entries: map[string]sourcetype.SourceData{
			"test-sig-1": {
				Signature: "test-sig-1",
				Source:    "test-source",
			},
		},
	}
	ctrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, mockRes, nil, secret, logger)

	now := time.Now()
	validToken, err := generateTestToken(secret, &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	t.Run("getSignature unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/item/test-sig-1", nil)
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("getSignature success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/item/test-sig-1", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		var data sourcetype.SourceData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if data.Signature != "test-sig-1" || data.Source != "test-source" {
			t.Errorf("unexpected data returned: %+v", data)
		}
	})

	t.Run("getSignature not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/item/non-existing", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("getSignature internal error", func(t *testing.T) {
		errResolver := &mockResolver{err: errors.New("database failure")}
		errCtrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, errResolver, nil, secret, logger)

		req := httptest.NewRequest(http.MethodGet, "/rest/item/test-sig-1", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		errCtrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("updateSignature unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/rest/item/test-sig-2", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("updateSignature invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/rest/item/test-sig-2", bytes.NewBufferString(`invalid json`))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("updateSignature success", func(t *testing.T) {
		item := sourcetype.SourceData{
			Signature: "test-sig-2",
			Source:    "updated-source",
		}
		itemBytes, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/rest/item/test-sig-2", bytes.NewBuffer(itemBytes))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "object test-sig-2 stored") {
			t.Errorf("expected body to contain 'object test-sig-2 stored', got: %s", w.Body.String())
		}
		if mockRes.entries["test-sig-2"].Source != "updated-source" {
			t.Errorf("expected resolver entry to be stored, got: %+v", mockRes.entries["test-sig-2"])
		}
	})

	t.Run("updateSignature internal error", func(t *testing.T) {
		errResolver := &mockResolver{err: errors.New("database failure")}
		errCtrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, errResolver, nil, secret, logger)

		item := sourcetype.SourceData{
			Signature: "test-sig-2",
		}
		itemBytes, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/rest/item/test-sig-2", bytes.NewBuffer(itemBytes))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		errCtrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
