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
	serverResolver := resolver.NewElasticResolver(elastic, conf.ElasticSearch.Index, conf.Client, conf.ElasticSearch.RoleWeights, logger)
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
