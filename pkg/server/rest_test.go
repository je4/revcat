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
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

type mockResolver struct {
	entries        map[string]sourcetype.SourceData
	err            error
	searchResult   *model.SearchResult
	searchByClient map[string]*model.SearchResult
}

func (m *mockResolver) Search(ctx context.Context, searchType string, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string, sort []*model.SortField) (*model.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.searchByClient != nil {
		client, _ := ctx.Value("client").(string)
		if res, ok := m.searchByClient[client]; ok {
			return res, nil
		}
	}
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return &model.SearchResult{}, nil
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

func (m *mockResolver) DeleteEntry(ctx context.Context, signature string) error {
	if m.err != nil {
		return m.err
	}
	if m.entries != nil {
		delete(m.entries, signature)
	}
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
		if paths["/search/{query}"] == nil {
			t.Errorf("expected doc.json paths to contain '/search/{query}', got %v", paths)
		}
		itemPath, ok := paths["/item/{signature}"].(map[string]any)
		if !ok {
			t.Fatalf("expected paths['/item/{signature}'] to be an object, got %T", paths["/item/{signature}"])
		}
		for _, method := range []string{"get", "post", "delete"} {
			if itemPath[method] == nil {
				t.Errorf("expected paths['/item/{signature}'] to contain method %q", method)
			}
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

	t.Run("deleteSignature unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/rest/item/test-sig-1", nil)
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("deleteSignature success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/rest/item/test-sig-1", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "object test-sig-1 deleted") {
			t.Errorf("expected body to contain 'object test-sig-1 deleted', got: %s", w.Body.String())
		}
		if _, ok := mockRes.entries["test-sig-1"]; ok {
			t.Errorf("expected entry test-sig-1 to be deleted from resolver, but it still exists")
		}
	})

	t.Run("deleteSignature internal error", func(t *testing.T) {
		errResolver := &mockResolver{err: errors.New("database failure")}
		errCtrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, errResolver, nil, secret, logger)

		req := httptest.NewRequest(http.MethodDelete, "/rest/item/test-sig-1", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		errCtrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestSearchMarkdownEndpoints(t *testing.T) {
	logger := newTestLogger()
	secret := "test-sync-key"

	strPtr := func(s string) *string {
		return &s
	}

	perfClient := &config.Client{
		Name: "performance",
		FieldWeights: map[string]map[string]float64{
			"type": {"performance": 5.0},
			"[persons].role": {
				"performer":   4.0,
				"contributor": 2.0,
			},
		},
		AND: []config.ClientANDQuery{
			{
				OR: []config.ClientOrQuery{
					{
						Field:  "category.keyword",
						Values: []string{"zotero2!!PCB_Basel"},
					},
				},
			},
		},
	}

	resTarget := &model.SearchResult{
		TotalCount: 2,
		Edges: []*model.MediathekFullEntry{
			{
				ID: "sig-perf-1",
				Base: &model.MediathekBaseEntry{
					Signature: "sig-perf-1",
					Type:      strPtr("performance"),
					Title:     []*model.MultiLangString{{Value: "Performance Mathis"}},
					Person: []*model.Person{
						{Name: "Muda Mathis", Role: strPtr("performer")},
					},
					Category: []string{"zotero2!!PCB_Basel"},
				},
			},
			{
				ID: "sig-book-2",
				Base: &model.MediathekBaseEntry{
					Signature: "sig-book-2",
					Type:      strPtr("book"),
					Title:     []*model.MultiLangString{{Value: "Book Mathis"}},
					Person: []*model.Person{
						{Name: "Muda Mathis", Role: strPtr("contributor")},
					},
					Category: []string{"zotero2!!PCB_Basel"},
				},
			},
		},
	}

	resBaseline := &model.SearchResult{
		TotalCount: 2,
		Edges: []*model.MediathekFullEntry{
			{
				ID: "sig-book-2",
				Base: &model.MediathekBaseEntry{
					Signature: "sig-book-2",
					Type:      strPtr("book"),
					Title:     []*model.MultiLangString{{Value: "Book Mathis"}},
					Person: []*model.Person{
						{Name: "Muda Mathis", Role: strPtr("contributor")},
					},
					Category: []string{"zotero2!!PCB_Basel"},
				},
			},
			{
				ID: "sig-perf-1",
				Base: &model.MediathekBaseEntry{
					Signature: "sig-perf-1",
					Type:      strPtr("performance"),
					Title:     []*model.MultiLangString{{Value: "Performance Mathis"}},
					Person: []*model.Person{
						{Name: "Muda Mathis", Role: strPtr("performer")},
					},
					Category: []string{"zotero2!!PCB_Basel"},
				},
			},
		},
	}

	mockRes := &mockResolver{
		searchByClient: map[string]*model.SearchResult{
			"performance":      resTarget,
			"default_baseline": resBaseline,
		},
	}

	ctrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, mockRes, []*config.Client{perfClient}, secret, logger)

	now := time.Now()
	validToken, err := generateTestToken(secret, &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	t.Run("search unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/search/muda%20mathis", nil)
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("search empty query bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/search/", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("search success with markdown output", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/search/muda%20mathis", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/markdown") {
			t.Errorf("expected Content-Type text/markdown, got: %s", contentType)
		}

		body := w.Body.String()
		expectedSnippets := []string{
			`# Search Prioritization & Ranking Matrix: "muda mathis"`,
			`**Query**: ` + "`muda mathis`",
			`**Client**: ` + "`performance`",
			`**Baseline**: ` + "`default_baseline`",
			`| Rank | Base # | Delta | Signature | Title | Type (Boost) | Role (Boost) | Max Boost | Tier | Status |`,
			`| [#01] | #02 | +1 | ` + "`sig-perf-1`" + ` | Performance Mathis | performance (x5.0) | performer (x4.0) | x5.0 | Tier 1 [4-5x] | BOOSTED (+Δ) |`,
			`| [#02] | #01 | -1 | ` + "`sig-book-2`" + ` | Book Mathis | book (x1.0) | contributor (x2.0) | x2.0 | Tier 2 [1.5-2x] | BOOSTED (-Δ) |`,
			`### Tier Distribution & Statistical Analysis`,
			`Tier 1 (High Boost 4.0-5.0x)`,
			`Tier 2 (Med Boost 1.5-2.0x)`,
			`### Prioritization & Filter Invariants`,
			`**Top 10 Boosted Dominance**`,
			`**Category Filter Enforcement**: PASS`,
		}

		for _, snippet := range expectedSnippets {
			if !strings.Contains(body, snippet) {
				t.Errorf("expected body to contain snippet:\n%s\n\nGot body:\n%s", snippet, body)
			}
		}
	})

	t.Run("search with query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/search/?q=muda%20mathis&client=performance&baseline=default_baseline&limit=10&groups=custom_group", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		ctrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "**Evaluated Hits Limit**: `10`") {
			t.Errorf("expected evaluated limit 10 in output, got: %s", w.Body.String())
		}
	})

	t.Run("search internal resolver error", func(t *testing.T) {
		errResolver := &mockResolver{err: errors.New("resolver search failed")}
		errCtrl := NewController("localhost:8080", "http://localhost:8080/graphql", nil, errResolver, []*config.Client{perfClient}, secret, logger)

		req := httptest.NewRequest(http.MethodGet, "/rest/search/muda%20mathis", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", validToken))
		w := httptest.NewRecorder()
		errCtrl.srv.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
