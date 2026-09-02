package client

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/revcat/v2/pkg/server"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"go.ub.unibas.ch/metastring/pkg/metaString"
)

func loadEnvFromFiles() {
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				v = strings.Trim(v, `"'`)
				if _, exists := os.LookupEnv(k); !exists {
					_ = os.Setenv(k, v)
				}
			}
		}
		break
	}
}

func TestGlobalElasticsearchClientService(t *testing.T) {
	// Attempt loading environment variables from .env if present
	loadEnvFromFiles()

	// 1. Load configuration from config/revcat.toml
	conf := &config.RevCatConfig{
		ElasticSearch: config.ElasticSearchConfig{
			Debug: false,
		},
	}
	if err := config.LoadRevCatConfig(config.ConfigFS, "revcat.toml", conf); err != nil {
		t.Fatalf("failed to load revcat.toml configuration: %v", err)
	}

	// 2. Fetch Elasticsearch API key live from the environment variable (never written to file)
	apiKey := os.Getenv("ELASTIC_APIKEY")
	if apiKey == "" {
		t.Skip("skipping test: ELASTIC_APIKEY environment variable is not set")
	}

	// 3. Initialize Elasticsearch typed client
	elasticConfig := elasticsearch.Config{
		Addresses: conf.ElasticSearch.Endpoint,
		APIKey:    apiKey,
	}
	elastic, err := elasticsearch.NewTypedClient(elasticConfig)
	if err != nil {
		t.Skipf("skipping test: cannot create typed elasticsearch client: %v", err)
	}

	// 4. Check if Elasticsearch endpoint is reachable
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if _, err := elastic.Info().Do(ctx); err != nil {
		t.Skipf("skipping test: cannot contact elasticsearch at %v: %v", conf.ElasticSearch.Endpoint, err)
	}

	// 5. Initialize Resolver and Server Controller
	logger := newTestLogger()
	serverResolver := resolver.NewElasticResolver(elastic, conf.ElasticSearch.Index, conf.Client, conf.ElasticSearch.RoleWeights, conf.ElasticSearch.FieldWeights, logger)
	syncJWTKey := "integration-test-sync-jwt-key"
	ctrl := server.NewController("localhost:0", "http://localhost:0/graphql", nil, serverResolver, conf.Client, syncJWTKey, logger)
	ts := httptest.NewServer(ctrl.Handler())
	defer ts.Close()

	// 6. Initialize REST Client
	cli, err := New(ts.URL, WithJWTKey(syncJWTKey))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	testSignature := fmt.Sprintf("global-test-%d", time.Now().UnixNano())
	testData := &sourcetype.SourceData{
		Signature: testSignature,
		Source:    "global-integration-test-source",
		Place:     "Basel",
		Date:      "2026-08-28",
	}

	// Ensure cleanup if test fails midway
	t.Cleanup(func() {
		_ = serverResolver.DeleteEntry(context.Background(), testSignature)
	})

	// Test 1: Schreiben (Write)
	t.Run("schreiben (write)", func(t *testing.T) {
		if err := cli.UpdateItem(ctx, testSignature, testData); err != nil {
			t.Fatalf("UpdateItem failed: %v", err)
		}
	})

	// Test 2: Lesen (Read)
	t.Run("lesen (read)", func(t *testing.T) {
		item, err := cli.GetItem(ctx, testSignature)
		if err != nil {
			t.Fatalf("GetItem failed: %v", err)
		}
		if item == nil {
			t.Fatal("expected non-nil item, got nil")
		}
		if item.Signature != testSignature {
			t.Errorf("expected signature %q, got %q", testSignature, item.Signature)
		}
		if item.Source != testData.Source {
			t.Errorf("expected source %q, got %q", testData.Source, item.Source)
		}
		if item.Place != testData.Place {
			t.Errorf("expected place %q, got %q", testData.Place, item.Place)
		}
	})

	// Test 3: Aktualisieren (Update)
	t.Run("aktualisieren (update)", func(t *testing.T) {
		updatedData := &sourcetype.SourceData{
			Signature: testSignature,
			Source:    "global-integration-test-source-updated",
			Place:     "Zürich",
			Date:      "2026-08-29",
		}
		if err := cli.UpdateItem(ctx, testSignature, updatedData); err != nil {
			t.Fatalf("UpdateItem failed: %v", err)
		}

		// Read back and verify updated fields
		item, err := cli.GetItem(ctx, testSignature)
		if err != nil {
			t.Fatalf("GetItem after update failed: %v", err)
		}
		if item == nil {
			t.Fatal("expected non-nil item, got nil")
		}
		if item.Signature != testSignature {
			t.Errorf("expected signature %q, got %q", testSignature, item.Signature)
		}
		if item.Source != updatedData.Source {
			t.Errorf("expected source %q, got %q", updatedData.Source, item.Source)
		}
		if item.Place != updatedData.Place {
			t.Errorf("expected place %q, got %q", updatedData.Place, item.Place)
		}
		if item.Date != updatedData.Date {
			t.Errorf("expected date %q, got %q", updatedData.Date, item.Date)
		}
	})

	// Test 4: Löschen (Delete)
	t.Run("loeschen (delete)", func(t *testing.T) {
		if err := cli.DeleteItem(ctx, testSignature); err != nil {
			t.Fatalf("DeleteItem failed: %v", err)
		}

		// Verify deletion
		itemAfterDelete, err := cli.GetItem(ctx, testSignature)
		if err == nil {
			t.Fatalf("expected error reading deleted item, but got item: %+v", itemAfterDelete)
		}
	})
}

func TestSearchRoleScoreFunctionIntegration(t *testing.T) {
	// 1. Attempt loading environment variables from .env if present
	loadEnvFromFiles()

	// 2. Load configuration from config/revcat.toml
	conf := &config.RevCatConfig{
		ElasticSearch: config.ElasticSearchConfig{
			Debug: false,
		},
	}
	if err := config.LoadRevCatConfig(config.ConfigFS, "revcat.toml", conf); err != nil {
		t.Fatalf("failed to load revcat.toml configuration: %v", err)
	}

	// 3. Fetch Elasticsearch API key live from the environment variable
	apiKey := os.Getenv("ELASTIC_APIKEY")
	if apiKey == "" {
		t.Skip("skipping test: ELASTIC_APIKEY environment variable is not set")
	}

	// 4. Initialize Elasticsearch typed client
	elasticConfig := elasticsearch.Config{
		Addresses: conf.ElasticSearch.Endpoint,
		APIKey:    apiKey,
	}
	elastic, err := elasticsearch.NewTypedClient(elasticConfig)
	if err != nil {
		t.Skipf("skipping test: cannot create typed elasticsearch client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := elastic.Info().Do(ctx); err != nil {
		t.Skipf("skipping test: cannot contact elasticsearch at %v: %v", conf.ElasticSearch.Endpoint, err)
	}

	// 5. Configure clients: one with role weights (author boosted) and one without role weights
	clientWithRoles := &config.Client{
		Name:   "client_with_roles",
		Groups: []string{"global/guest"},
		RoleWeights: map[string]float64{
			"author":      50.0,
			"contributor": 1.0,
		},
	}
	clientWithoutRoles := &config.Client{
		Name:        "client_without_roles",
		Groups:      []string{"global/guest"},
		RoleWeights: map[string]float64{},
	}
	clients := []*config.Client{clientWithRoles, clientWithoutRoles}

	logger := newTestLogger()
	testResolver := resolver.NewElasticResolver(elastic, conf.ElasticSearch.Index, clients, nil, nil, logger)

	// 6. Create test documents
	nowNano := time.Now().UnixNano()
	targetName := fmt.Sprintf("RoleScoreTarget_%d", nowNano)
	sigAuthorDoc := fmt.Sprintf("role-test-author-%d", nowNano)
	sigContribDoc := fmt.Sprintf("role-test-contrib-%d", nowNano)

	// Doc A: role is "author", title is a standard generic title (matches query ONLY via persons.name)
	docAuthor := &sourcetype.SourceData{
		Signature: sigAuthorDoc,
		Source:    "role-integration-test-source",
		Persons: []sourcetype.Person{
			{
				Name: targetName,
				Role: "author",
			},
		},
		ACL: map[string][]string{
			"meta":    {"global/guest"},
			"content": {"global/guest"},
		},
		Category: []string{"zotero2!!PCB_Basel"},
	}
	_ = docAuthor.SetTitle(metaString.NewMetaString("Generic Document Title Alpha"))

	// Doc B: role is "contributor", title explicitly contains targetName (matches query in BOTH title AND persons.name -> higher base score)
	docContrib := &sourcetype.SourceData{
		Signature: sigContribDoc,
		Source:    "role-integration-test-source",
		Persons: []sourcetype.Person{
			{
				Name: targetName,
				Role: "contributor",
			},
		},
		ACL: map[string][]string{
			"meta":    {"global/guest"},
			"content": {"global/guest"},
		},
		Category: []string{"zotero2!!PCB_Basel"},
	}
	_ = docContrib.SetTitle(metaString.NewMetaString(fmt.Sprintf("Explicit Match %s in Title", targetName)))

	// 7. Cleanup after test completion
	t.Cleanup(func() {
		_ = testResolver.DeleteEntry(context.Background(), sigAuthorDoc)
		_ = testResolver.DeleteEntry(context.Background(), sigContribDoc)
	})

	// Store both entries into Elasticsearch
	if err := testResolver.StoreEntry(ctx, sigAuthorDoc, docAuthor); err != nil {
		t.Fatalf("failed to store docAuthor: %v", err)
	}
	if err := testResolver.StoreEntry(ctx, sigContribDoc, docContrib); err != nil {
		t.Fatalf("failed to store docContrib: %v", err)
	}

	// Wait for Elasticsearch indexing refresh interval
	time.Sleep(2 * time.Second)

	// 8. Search without role scoring
	ctxWithoutRoles := context.WithValue(ctx, "groups", []string{"global/guest"})
	ctxWithoutRoles = context.WithValue(ctxWithoutRoles, "client", "client_without_roles")

	t.Logf("Searching for query %q", targetName)
	resWithoutRoles, err := testResolver.Search(
		ctxWithoutRoles,
		"all",
		targetName,
		nil, nil, nil, nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("search without role scoring failed: %v", err)
	}
	t.Logf("TotalCount without roles: %d, Edges len: %d", resWithoutRoles.TotalCount, len(resWithoutRoles.Edges))

	// 9. Search with role scoring
	ctxWithRoles := context.WithValue(ctx, "groups", []string{"global/guest"})
	ctxWithRoles = context.WithValue(ctxWithRoles, "client", "client_with_roles")

	resWithRoles, err := testResolver.Search(
		ctxWithRoles,
		"all",
		targetName,
		nil, nil, nil, nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("search with role scoring failed: %v", err)
	}

	// 10. Verify results
	if len(resWithoutRoles.Edges) < 2 {
		t.Fatalf("expected at least 2 results without role scoring, got %d", len(resWithoutRoles.Edges))
	}
	if len(resWithRoles.Edges) < 2 {
		t.Fatalf("expected at least 2 results with role scoring, got %d", len(resWithRoles.Edges))
	}

	firstWithout := resWithoutRoles.Edges[0].Base.Signature
	secondWithout := resWithoutRoles.Edges[1].Base.Signature
	firstWith := resWithRoles.Edges[0].Base.Signature
	secondWith := resWithRoles.Edges[1].Base.Signature

	t.Logf("Result ranking without role score function: 1st=%s, 2nd=%s", firstWithout, secondWithout)
	t.Logf("Result ranking with role score function:    1st=%s, 2nd=%s", firstWith, secondWith)

	// Without role weights, docContrib ranks higher due to title match
	if firstWithout != sigContribDoc {
		t.Errorf("expected docContrib (%s) to rank 1st without role score function, got %s", sigContribDoc, firstWithout)
	}

	// With role weights, docAuthor gets boosted by author role weight and ranks higher
	if firstWith != sigAuthorDoc {
		t.Errorf("expected docAuthor (%s) to rank 1st with role score function, got %s", sigAuthorDoc, firstWith)
	}

	// Confirm that the result ranking differs
	if firstWithout == firstWith {
		t.Errorf("expected role score function to alter ranking order, but top result was identical: %s", firstWith)
	}
}
