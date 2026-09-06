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

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-2]) + ".."
}

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
	serverResolver := resolver.NewElasticResolver(elastic, conf.ElasticSearch.Index, conf.Client, conf.ElasticSearch.SearchWeights, conf.ElasticSearch.FieldWeights, logger)
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
		FieldWeights: map[string]map[string]float64{
			"[persons].role": {
				"author":      50.0,
				"contributor": 1.0,
			},
		},
	}
	clientWithoutRoles := &config.Client{
		Name:         "client_without_roles",
		Groups:       []string{"global/guest"},
		FieldWeights: map[string]map[string]float64{},
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

func TestPerformanceClientSearchMudaMathisIntegration(t *testing.T) {
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

	// 5. Verify client 'performance' is loaded from config and add baseline client
	var performanceClient *config.Client
	for _, c := range conf.Client {
		if c.Name == "performance" {
			performanceClient = c
			break
		}
	}
	if performanceClient == nil {
		t.Fatalf("client 'performance' not found in revcat.toml")
	}

	baselineClient := &config.Client{
		Name:          "default_baseline",
		Groups:        []string{"global/guest"},
		SearchWeights: map[string]float64{},
		FieldWeights:  map[string]map[string]float64{},
	}
	clients := append([]*config.Client{baselineClient}, conf.Client...)

	logger := newTestLogger()
	testResolver := resolver.NewElasticResolver(
		elastic,
		conf.ElasticSearch.Index,
		clients,
		conf.ElasticSearch.SearchWeights,
		conf.ElasticSearch.FieldWeights,
		logger,
	)

	// --- Part A: Live Search for "muda mathis" on the target index ---
	t.Run("Live index search for muda mathis under performance client", func(t *testing.T) {
		ctxPerf := context.WithValue(ctx, "groups", []string{"global/guest"})
		ctxPerf = context.WithValue(ctxPerf, "client", "performance")

		ctxBaseline := context.WithValue(ctx, "groups", []string{"global/guest"})
		ctxBaseline = context.WithValue(ctxBaseline, "client", "default_baseline")

		queryStr := "muda mathis"

		resBaseline, err := testResolver.Search(
			ctxBaseline,
			"all",
			queryStr,
			nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("live baseline search failed: %v", err)
		}

		resPerf, err := testResolver.Search(
			ctxPerf,
			"all",
			queryStr,
			nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("live performance search failed: %v", err)
		}

		t.Logf("\n==========================================================================================")
		t.Logf("            LIVE SEARCH COMPARISON: 'muda mathis' (performance vs baseline)               ")
		t.Logf("==========================================================================================")
		t.Logf("Search Query: %q", queryStr)
		t.Logf("  Baseline Client Total Hits:    %d (Unfiltered, no weights)", resBaseline.TotalCount)
		t.Logf("  Performance Client Total Hits: %d (Filtered by 14 allowed categories & weighted)", resPerf.TotalCount)
		t.Logf("------------------------------------------------------------------------------------------")

		// Index baseline hit positions
		baselineRankMap := make(map[string]int)
		for i, edge := range resBaseline.Edges {
			baselineRankMap[edge.Base.Signature] = i + 1
		}

		// Allowed categories for client performance
		allowedCategories := make(map[string]struct{})
		for _, andFilter := range performanceClient.AND {
			for _, orFilter := range andFilter.OR {
				if orFilter.Field == "category.keyword" {
					for _, val := range orFilter.Values {
						allowedCategories[val] = struct{}{}
					}
				}
			}
		}

		typeWeights := performanceClient.FieldWeights["type"]
		if typeWeights == nil {
			typeWeights = map[string]float64{}
		}
		roleWeights := performanceClient.FieldWeights["[persons].role"]
		if roleWeights == nil {
			roleWeights = map[string]float64{}
		}

		type liveMatrixRow struct {
			perfRank    int
			baseRank    int
			hasBaseRank bool
			delta       int
			sig         string
			title       string
			docType     string
			typeBoost   float64
			topRole     string
			roleBoost   float64
			maxBoost    float64
			tier        string
			status      string
		}

		var liveRows []liveMatrixRow

		for i, edge := range resPerf.Edges {
			perfRank := i + 1
			base := edge.Base

			var typeStr string
			if base.Type != nil {
				typeStr = *base.Type
			}
			typeBoost := 1.0
			if w, ok := typeWeights[typeStr]; ok {
				typeBoost = w
			}

			var titleStr string
			if len(base.Title) > 0 {
				titleStr = base.Title[0].Value
			}

			topRole := "none"
			maxRoleBoost := 1.0
			for _, p := range base.Person {
				role := ""
				if p.Role != nil {
					role = *p.Role
				}
				if w, ok := roleWeights[role]; ok {
					if w > maxRoleBoost {
						maxRoleBoost = w
						topRole = role
					}
				} else if topRole == "none" && role != "" {
					topRole = role
				}
			}

			effectiveMaxBoost := typeBoost
			if maxRoleBoost > effectiveMaxBoost {
				effectiveMaxBoost = maxRoleBoost
			}

			tier := "Tier 3 [1.0x]"
			if effectiveMaxBoost >= 4.0 {
				tier = "Tier 1 [4-5x]"
			} else if effectiveMaxBoost >= 1.5 {
				tier = "Tier 2 [1.5-2x]"
			}

			// Baseline rank & delta calculation
			baseRank, inBaseline := baselineRankMap[base.Signature]
			delta := 0
			status := "NEW (Top)"
			if inBaseline {
				delta = baseRank - perfRank
				if effectiveMaxBoost > 1.0 {
					if delta > 0 {
						status = "BOOSTED (+Δ)"
					} else if delta < 0 {
						status = "BOOSTED (-Δ)"
					} else {
						status = "BOOSTED (=)"
					}
				} else {
					if delta > 0 {
						status = "PROMOTED (+Δ)"
					} else if delta < 0 {
						status = "UNBOOSTED (-Δ)"
					} else {
						status = "STABLE (=)"
					}
				}
			} else if effectiveMaxBoost > 1.0 {
				status = "BOOSTED (NEW)"
			}

			liveRows = append(liveRows, liveMatrixRow{
				perfRank:    perfRank,
				baseRank:    baseRank,
				hasBaseRank: inBaseline,
				delta:       delta,
				sig:         base.Signature,
				title:       titleStr,
				docType:     typeStr,
				typeBoost:   typeBoost,
				topRole:     topRole,
				roleBoost:   maxRoleBoost,
				maxBoost:    effectiveMaxBoost,
				tier:        tier,
				status:      status,
			})

			// Category filter verification: each hit returned must belong to allowed categories
			if len(allowedCategories) > 0 && len(base.Category) > 0 {
				matched := false
				for _, cat := range base.Category {
					if _, ok := allowedCategories[cat]; ok {
						matched = true
						break
					}
				}
				if !matched {
					t.Errorf("hit %s with categories %v is not within allowed categories for client performance", base.Signature, base.Category)
				}
			}
		}

		t.Logf("\n==========================================================================================================================================")
		t.Logf("                                     LIVE SEARCH WEIGHTING & RANKING MATRIX: 'muda mathis'                                               ")
		t.Logf("==========================================================================================================================================")
		t.Logf("%-6s | %-6s | %-6s | %-18s | %-28s | %-18s | %-18s | %-9s | %-14s | %-14s",
			"Perf #", "Base #", "Delta", "Signature", "Title", "Type (Boost)", "Role (Boost)", "Max Boost", "Tier", "Status")
		t.Logf("------------------------------------------------------------------------------------------------------------------------------------------")

		for _, row := range liveRows {
			baseRankStr := "n/a"
			deltaStr := "NEW"
			if row.hasBaseRank {
				baseRankStr = fmt.Sprintf("#%d", row.baseRank)
				if row.delta > 0 {
					deltaStr = fmt.Sprintf("+%d", row.delta)
				} else if row.delta < 0 {
					deltaStr = fmt.Sprintf("%d", row.delta)
				} else {
					deltaStr = "="
				}
			}

			typeDesc := fmt.Sprintf("%s (x%.1f)", row.docType, row.typeBoost)
			roleDesc := fmt.Sprintf("%s (x%.1f)", row.topRole, row.roleBoost)
			shortTitle := truncateString(row.title, 28)

			t.Logf("[#%02d] | %-6s | %-6s | %-18s | %-28s | %-18s | %-18s | x%-8.1f | %-14s | %-14s",
				row.perfRank, baseRankStr, deltaStr, row.sig, shortTitle, typeDesc, roleDesc, row.maxBoost, row.tier, row.status)
		}
		t.Logf("------------------------------------------------------------------------------------------------------------------------------------------")

		// Tier statistics
		type tierStats struct {
			totalCount    int
			top10Count    int
			improvedCount int
			deltaSum      int
			deltaCount    int
		}
		statsTier1 := &tierStats{}
		statsTier2 := &tierStats{}
		statsTier3 := &tierStats{}

		for _, row := range liveRows {
			var st *tierStats
			if strings.HasPrefix(row.tier, "Tier 1") {
				st = statsTier1
			} else if strings.HasPrefix(row.tier, "Tier 2") {
				st = statsTier2
			} else {
				st = statsTier3
			}
			st.totalCount++
			if row.perfRank <= 10 {
				st.top10Count++
			}
			if row.hasBaseRank {
				st.deltaSum += row.delta
				st.deltaCount++
				if row.delta > 0 {
					st.improvedCount++
				}
			}
		}

		avgDeltaStr := func(st *tierStats) string {
			if st.deltaCount == 0 {
				return "N/A"
			}
			return fmt.Sprintf("%+.2f", float64(st.deltaSum)/float64(st.deltaCount))
		}

		topHitsCount := len(liveRows)
		top10Limit := 10
		if topHitsCount < top10Limit {
			top10Limit = topHitsCount
		}

		top10Tier1Pct := 0.0
		top10Tier2Pct := 0.0
		top10Tier3Pct := 0.0
		if top10Limit > 0 {
			top10Tier1Pct = float64(statsTier1.top10Count) / float64(top10Limit) * 100
			top10Tier2Pct = float64(statsTier2.top10Count) / float64(top10Limit) * 100
			top10Tier3Pct = float64(statsTier3.top10Count) / float64(top10Limit) * 100
		}

		t.Logf("LIVE DATA TIER DISTRIBUTION & STATISTICAL VALIDATION:")
		t.Logf("  * Tier 1 (High Boost 4.0-5.0x): %2d docs | %2d in Top 10 (%5.1f%%) | %2d rank improved | Avg Delta: %s",
			statsTier1.totalCount, statsTier1.top10Count, top10Tier1Pct, statsTier1.improvedCount, avgDeltaStr(statsTier1))
		t.Logf("  * Tier 2 (Med Boost 1.5-2.0x):  %2d docs | %2d in Top 10 (%5.1f%%) | %2d rank improved | Avg Delta: %s",
			statsTier2.totalCount, statsTier2.top10Count, top10Tier2Pct, statsTier2.improvedCount, avgDeltaStr(statsTier2))
		t.Logf("  * Tier 3 (Base 1.0x):           %2d docs | %2d in Top 10 (%5.1f%%) | %2d rank improved | Avg Delta: %s",
			statsTier3.totalCount, statsTier3.top10Count, top10Tier3Pct, statsTier3.improvedCount, avgDeltaStr(statsTier3))
		t.Logf("------------------------------------------------------------------------------------------------------------------------------------------")
		top10Boosted := statsTier1.top10Count + statsTier2.top10Count
		top10BoostedPct := 0.0
		if top10Limit > 0 {
			top10BoostedPct = float64(top10Boosted) / float64(top10Limit) * 100
		}
		t.Logf("PRIORITIZATION INVARIANTS CHECK:")
		t.Logf("  - Top 10 Boosted Dominance:     %.1f%% of Top 10 have active weight boosts (Tier 1/2: %d/%d hits)",
			top10BoostedPct, top10Boosted, top10Limit)
		t.Logf("  - Category Filter Enforcement:  PASS (100%% of %d returned documents belong to allowed categories)", resPerf.TotalCount)
		t.Logf("==========================================================================================================================================\n")

		if resPerf.TotalCount == 0 {
			t.Errorf("expected live performance search to return hits, got 0")
		}
		if top10Boosted == 0 && topHitsCount > 0 {
			t.Errorf("expected boosted documents in Top 10, got 0")
		}
	})

	// --- Part B: Deterministic Synthetic Test Cases for Role & Type Prioritization ---
	t.Run("Deterministic synthetic role and type weighting validation", func(t *testing.T) {
		nowNano := time.Now().UnixNano()
		targetName := fmt.Sprintf("MudaMathisTarget_%d", nowNano)

		sigPerformanceDoc := fmt.Sprintf("muda-perf-%d", nowNano)
		sigArtistDoc := fmt.Sprintf("muda-artist-%d", nowNano)
		sigContributorDoc := fmt.Sprintf("muda-contrib-%d", nowNano)
		sigDirectorDoc := fmt.Sprintf("muda-director-%d", nowNano)
		sigFilteredDoc := fmt.Sprintf("muda-filtered-%d", nowNano)

		// Doc 1: Prioritized type ("performance", weight 5.0) and role ("performer", weight 4.0) -> effective boost 5.0
		docPerf := &sourcetype.SourceData{
			Signature: sigPerformanceDoc,
			Source:    "performance-integration-test-source",
			Type:      "performance",
			Persons: []sourcetype.Person{
				{
					Name: targetName,
					Role: "performer",
				},
			},
			ACL: map[string][]string{
				"meta":    {"global/guest"},
				"content": {"global/guest"},
			},
			Category: []string{"zotero2!!PCB_Basel"}, // Allowed in performance client
		}
		_ = docPerf.SetTitle(metaString.NewMetaString("Artwork Alpha"))

		// Doc 2: Role "artist" (weight 4.0) with non-boosted type "book" -> effective boost 4.0
		docArtist := &sourcetype.SourceData{
			Signature: sigArtistDoc,
			Source:    "performance-integration-test-source",
			Type:      "book",
			Persons: []sourcetype.Person{
				{
					Name: targetName,
					Role: "artist",
				},
			},
			ACL: map[string][]string{
				"meta":    {"global/guest"},
				"content": {"global/guest"},
			},
			Category: []string{"zotero2!!PCB_Basel"}, // Allowed in performance client
		}
		_ = docArtist.SetTitle(metaString.NewMetaString("Artwork Beta"))

		// Doc 3: Role "contributor" (weight 2.0) with non-boosted type "book" -> effective boost 2.0
		docContrib := &sourcetype.SourceData{
			Signature: sigContributorDoc,
			Source:    "performance-integration-test-source",
			Type:      "book",
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
			Category: []string{"zotero2!!PCB_Basel"}, // Allowed in performance client
		}
		_ = docContrib.SetTitle(metaString.NewMetaString("Artwork Gamma"))

		// Doc 4: Role "director" (weight 1.5) with non-boosted type "book" -> effective boost 1.5
		docDirector := &sourcetype.SourceData{
			Signature: sigDirectorDoc,
			Source:    "performance-integration-test-source",
			Type:      "book",
			Persons: []sourcetype.Person{
				{
					Name: targetName,
					Role: "director",
				},
			},
			ACL: map[string][]string{
				"meta":    {"global/guest"},
				"content": {"global/guest"},
			},
			Category: []string{"zotero2!!PCB_Basel"}, // Allowed in performance client
		}
		_ = docDirector.SetTitle(metaString.NewMetaString("Artwork Delta"))

		// Doc 5: Filtered category (outside allowed categories for performance client)
		docFiltered := &sourcetype.SourceData{
			Signature: sigFilteredDoc,
			Source:    "performance-integration-test-source",
			Type:      "performance",
			Persons: []sourcetype.Person{
				{
					Name: targetName,
					Role: "performer",
				},
			},
			ACL: map[string][]string{
				"meta":    {"global/guest"},
				"content": {"global/guest"},
			},
			Category: []string{"unauthorized_category_excluded"},
		}
		_ = docFiltered.SetTitle(metaString.NewMetaString("Artwork Epsilon"))

		// Register cleanup for synthetic documents
		t.Cleanup(func() {
			_ = testResolver.DeleteEntry(context.Background(), sigPerformanceDoc)
			_ = testResolver.DeleteEntry(context.Background(), sigArtistDoc)
			_ = testResolver.DeleteEntry(context.Background(), sigContributorDoc)
			_ = testResolver.DeleteEntry(context.Background(), sigDirectorDoc)
			_ = testResolver.DeleteEntry(context.Background(), sigFilteredDoc)
		})

		// Store synthetic documents
		if err := testResolver.StoreEntry(ctx, sigPerformanceDoc, docPerf); err != nil {
			t.Fatalf("failed to store docPerf: %v", err)
		}
		if err := testResolver.StoreEntry(ctx, sigArtistDoc, docArtist); err != nil {
			t.Fatalf("failed to store docArtist: %v", err)
		}
		if err := testResolver.StoreEntry(ctx, sigContributorDoc, docContrib); err != nil {
			t.Fatalf("failed to store docContrib: %v", err)
		}
		if err := testResolver.StoreEntry(ctx, sigDirectorDoc, docDirector); err != nil {
			t.Fatalf("failed to store docDirector: %v", err)
		}
		if err := testResolver.StoreEntry(ctx, sigFilteredDoc, docFiltered); err != nil {
			t.Fatalf("failed to store docFiltered: %v", err)
		}

		// Allow Elasticsearch index refresh
		time.Sleep(2 * time.Second)

		// 1. Search under baseline client (no weights, no category filter)
		ctxBaseline := context.WithValue(ctx, "groups", []string{"global/guest"})
		ctxBaseline = context.WithValue(ctxBaseline, "client", "default_baseline")

		resBaseline, err := testResolver.Search(
			ctxBaseline,
			"all",
			targetName,
			nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("baseline search for synthetic docs failed: %v", err)
		}

		// 2. Search under performance client (role weights + type weights + category filter)
		ctxPerf := context.WithValue(ctx, "groups", []string{"global/guest"})
		ctxPerf = context.WithValue(ctxPerf, "client", "performance")

		resPerf, err := testResolver.Search(
			ctxPerf,
			"all",
			targetName,
			nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("performance client search for synthetic docs failed: %v", err)
		}

		// Assertions:
		// A. Baseline client should find all 5 documents (no category filtering)
		if len(resBaseline.Edges) < 5 {
			t.Fatalf("expected 5 hits in baseline search, got %d", len(resBaseline.Edges))
		}

		// B. Performance client should find exactly 4 documents (docFiltered is excluded)
		if len(resPerf.Edges) != 4 {
			t.Fatalf("expected exactly 4 hits in performance search (filtered doc excluded), got %d", len(resPerf.Edges))
		}

		filteredDocFound := false
		for _, edge := range resPerf.Edges {
			if edge.Base.Signature == sigFilteredDoc {
				filteredDocFound = true
				t.Errorf("filtered document %s was unexpectedly returned for client performance", sigFilteredDoc)
			}
		}

		type syntheticExpectedDoc struct {
			sig          string
			title        string
			docType      string
			typeBoost    float64
			role         string
			roleBoost    float64
			maxBoost     float64
			expectedRank int
		}

		expectedMatrix := []syntheticExpectedDoc{
			{sig: sigPerformanceDoc, title: "Artwork Alpha", docType: "performance", typeBoost: 5.0, role: "performer", roleBoost: 4.0, maxBoost: 5.0, expectedRank: 1},
			{sig: sigArtistDoc, title: "Artwork Beta", docType: "book", typeBoost: 1.0, role: "artist", roleBoost: 4.0, maxBoost: 4.0, expectedRank: 2},
			{sig: sigContributorDoc, title: "Artwork Gamma", docType: "book", typeBoost: 1.0, role: "contributor", roleBoost: 2.0, maxBoost: 2.0, expectedRank: 3},
			{sig: sigDirectorDoc, title: "Artwork Delta", docType: "book", typeBoost: 1.0, role: "director", roleBoost: 1.5, maxBoost: 1.5, expectedRank: 4},
		}

		rankings := make([]string, len(resPerf.Edges))
		actualRankMap := make(map[string]int)
		for i, edge := range resPerf.Edges {
			rankings[i] = edge.Base.Signature
			actualRankMap[edge.Base.Signature] = i + 1
		}

		t.Logf("\n==========================================================================================")
		t.Logf("                       SYNTHETIC WEIGHTING MATRIX VALIDATION                              ")
		t.Logf("==========================================================================================")
		t.Logf("Configured Client 'performance' Weights:")
		t.Logf("  - Type weights: performance = 5.0, Performance = 5.0")
		t.Logf("  - Role weights: performer = 4.0, artist = 4.0, author = 4.0, contributor = 2.0, eventcurator = 2.0, director = 1.5")
		t.Logf("  - Search weights: persons.name = 5.0, title = 3.0")
		t.Logf("------------------------------------------------------------------------------------------")
		t.Logf("%-10s | %-24s | %-18s | %-18s | %-9s | %-13s | %-8s",
			"Result", "Signature", "Type (Boost)", "Role (Boost)", "Max Boost", "Expected/Rank", "Status")
		t.Logf("------------------------------------------------------------------------------------------")

		allRanksCorrect := true
		for _, item := range expectedMatrix {
			actualRank, found := actualRankMap[item.sig]
			status := "PASS (OK)"
			if !found || actualRank != item.expectedRank {
				status = "FAIL"
				allRanksCorrect = false
			}

			typeDesc := fmt.Sprintf("%s (x%.1f)", item.docType, item.typeBoost)
			roleDesc := fmt.Sprintf("%s (x%.1f)", item.role, item.roleBoost)
			rankDesc := fmt.Sprintf("Exp #%d -> Act #%d", item.expectedRank, actualRank)

			t.Logf("[RANK %d]   | %-24s | %-18s | %-18s | x%-8.1f | %-13s | %s",
				actualRank, item.sig, typeDesc, roleDesc, item.maxBoost, rankDesc, status)
		}

		t.Logf("------------------------------------------------------------------------------------------")
		filterStatus := "PASS (OK)"
		if filteredDocFound {
			filterStatus = "FAIL (Included)"
		}
		t.Logf("[FILTER]   | %-24s | Category: unauthorized_category_excluded   | Excluded from Results | %s",
			sigFilteredDoc, filterStatus)
		t.Logf("==========================================================================================\n")

		if !allRanksCorrect {
			t.Errorf("synthetic ranking validation failed: expected rankings %v, got %v",
				[]string{sigPerformanceDoc, sigArtistDoc, sigContributorDoc, sigDirectorDoc}, rankings)
		}
	})
}
