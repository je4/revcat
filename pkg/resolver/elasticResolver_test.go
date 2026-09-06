package resolver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/revcat/v2/config"
)

type searchCaptureTransport func(*http.Request) (*http.Response, error)

func (f searchCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestElasticResolver_AddedBoost(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, query := range []string{"", "test"} {
			t.Run(strings.Join([]string{map[bool]string{false: "disabled", true: "enabled"}[enabled], query}, "/"), func(t *testing.T) {
				var body struct {
					Query map[string]json.RawMessage `json:"query"`
				}
				elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
					Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
							Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)),
						}, nil
					}),
				})
				if err != nil {
					t.Fatal(err)
				}
				client := &config.Client{Name: "test", AddedBoost: enabled, FieldWeights: map[string]map[string]float64{"[persons].role": {"author": 3}}}
				r := NewElasticResolver(elastic, "test", []*config.Client{client}, nil, nil, nil)
				ctx := context.WithValue(context.Background(), "client", client.Name)
				if _, err := r.Search(ctx, "all", query, nil, nil, nil, nil, nil, nil, nil); err != nil {
					t.Fatal(err)
				}
				if enabled {
					var score struct {
						Functions []struct {
							Filter struct {
								Exists struct {
									Field string `json:"field"`
								} `json:"exists"`
							} `json:"filter"`
							Gauss map[string]struct {
								Origin string  `json:"origin"`
								Scale  string  `json:"scale"`
								Offset string  `json:"offset"`
								Decay  float64 `json:"decay"`
							} `json:"gauss"`
							Weight float64 `json:"weight"`
						} `json:"functions"`
						ScoreMode string                     `json:"score_mode"`
						BoostMode string                     `json:"boost_mode"`
						Query     map[string]json.RawMessage `json:"query"`
					}
					if err := json.Unmarshal(body.Query["function_score"], &score); err != nil {
						t.Fatal(err)
					}
					if score.ScoreMode != "sum" || score.BoostMode != "sum" || len(score.Functions) != 3 {
						t.Fatalf("unexpected added boost: %+v", score)
					}
					for i, fn := range score.Functions[:2] {
						date := fn.Gauss["dateadded"]
						if fn.Filter.Exists.Field != "dateadded" || date.Origin != "now" || date.Scale != []string{"30d", "335d"}[i] || date.Offset != []string{"0d", "30d"}[i] || date.Decay != []float64{0.5, 0.1}[i] || fn.Weight != []float64{0.15, 0.10}[i] {
							t.Errorf("unexpected decay function %d: %+v", i, fn)
						}
					}
					if score.Functions[2].Weight != 0 || score.Functions[2].Filter.Exists.Field != "" || score.Functions[2].Gauss != nil {
						t.Error("expected unconditional zero fallback")
					}
					body.Query = score.Query
				}
				if query == "" {
					if _, ok := body.Query["bool"]; !ok {
						t.Error("expected original bool query")
					}
				} else if !strings.Contains(string(body.Query["function_score"]), `"boost_mode":"multiply"`) {
					t.Error("expected existing role boost to be preserved")
				}
			})
		}
	}
}

func TestParseFieldKey(t *testing.T) {
	tests := []struct {
		input             string
		expectedPath      string
		expectedFullField string
		expectedTermField string
	}{
		{
			input:             "[persons].role",
			expectedPath:      "persons",
			expectedFullField: "persons.role",
			expectedTermField: "persons.role.keyword",
		},
		{
			input:             "[media.audio].type",
			expectedPath:      "media.audio",
			expectedFullField: "media.audio.type",
			expectedTermField: "media.audio.type.keyword",
		},
		{
			input:             "[persons].identifier.url",
			expectedPath:      "persons",
			expectedFullField: "persons.identifier.url",
			expectedTermField: "persons.identifier.url.keyword",
		},
		{
			input:             "[persons].role.keyword",
			expectedPath:      "persons",
			expectedFullField: "persons.role.keyword",
			expectedTermField: "persons.role.keyword",
		},
		{
			input:             "category",
			expectedPath:      "",
			expectedFullField: "category",
			expectedTermField: "category.keyword",
		},
		{
			input:             "category.keyword",
			expectedPath:      "",
			expectedFullField: "category.keyword",
			expectedTermField: "category.keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, fullField, termField := parseFieldKey(tt.input)
			if path != tt.expectedPath {
				t.Errorf("parseFieldKey(%q) path = %q, want %q", tt.input, path, tt.expectedPath)
			}
			if fullField != tt.expectedFullField {
				t.Errorf("parseFieldKey(%q) fullField = %q, want %q", tt.input, fullField, tt.expectedFullField)
			}
			if termField != tt.expectedTermField {
				t.Errorf("parseFieldKey(%q) termField = %q, want %q", tt.input, termField, tt.expectedTermField)
			}
		})
	}
}

func TestLoadRevCatConfig_FieldWeights(t *testing.T) {
	conf := &config.RevCatConfig{}
	if err := config.LoadRevCatConfig(config.ConfigFS, "revcat.toml", conf); err != nil {
		t.Fatalf("failed to load revcat.toml: %v", err)
	}

	if len(conf.ElasticSearch.FieldWeights) == 0 {
		t.Fatalf("expected fieldweights to be parsed from revcat.toml, but got empty map")
	}

	globalPersonRoleWeights, ok := conf.ElasticSearch.FieldWeights["[persons].role"]
	if !ok {
		t.Fatalf("expected '[persons].role' to be present in ElasticSearch.FieldWeights")
	}

	expectedRoles := map[string]float64{
		"author":      3.0,
		"artist":      2.5,
		"director":    2.5,
		"performer":   2.5,
		"composer":    2.5,
		"Composer":    2.5,
		"camera":      2.0,
		"editor":      1.5,
		"creator":     2.0,
		"contributor": 1.2,
		"translator":  1.1,
	}

	for role, expectedWeight := range expectedRoles {
		weight, ok := globalPersonRoleWeights[role]
		if !ok {
			t.Errorf("expected role %q to be present in global [persons].role weights", role)
			continue
		}
		if weight != expectedWeight {
			t.Errorf("expected role %q to have weight %v, got %v", role, expectedWeight, weight)
		}
	}

	// Verify client-specific fieldweights
	var performanceClient, inkClient *config.Client
	for _, c := range conf.Client {
		if c.Name == "performance" {
			performanceClient = c
		}
		if c.Name == "ink" {
			inkClient = c
		}
	}

	if performanceClient == nil {
		t.Fatal("expected client 'performance' to exist in config")
	}
	perfRoles, ok := performanceClient.FieldWeights["[persons].role"]
	if !ok {
		t.Fatalf("expected client 'performance' to have '[persons].role' in fieldweights")
	}
	if perfRoles["artist"] != 4.0 || perfRoles["performer"] != 4.0 || perfRoles["director"] != 1.5 || perfRoles["author"] != 4.0 || perfRoles["eventcurator"] != 2.0 || perfRoles["contributor"] != 2.0 {
		t.Errorf("unexpected roleweights for client 'performance': %v", perfRoles)
	}

	if inkClient == nil {
		t.Fatal("expected client 'ink' to exist in config")
	}
	inkRoles, ok := inkClient.FieldWeights["[persons].role"]
	if !ok {
		t.Fatalf("expected client 'ink' to have '[persons].role' in fieldweights")
	}
	if inkRoles["author"] != 4.0 || inkRoles["editor"] != 2.0 || inkRoles["creator"] != 2.0 || inkRoles["translator"] != 1.5 || inkRoles["contributor"] != 1.2 {
		t.Errorf("unexpected roleweights for client 'ink': %v", inkRoles)
	}
}

func TestElasticResolver_FunctionScoreQueryGeneration(t *testing.T) {
	fieldWeights := map[string]map[string]float64{
		"[persons].role": {
			"Autor":      3.0,
			"Übersetzer": 1.1,
		},
		"[media.audio].type": {
			"interview": 2.0,
		},
		"category": {
			"art": 2.5,
		},
		"empty_weights": {
			"none": 0.0,
		},
	}

	var capturedBody struct {
		Query map[string]json.RawMessage `json:"query"`
	}

	elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatal(err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
				Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := &config.Client{
		Name:         "test_client",
		FieldWeights: fieldWeights,
	}

	r := NewElasticResolver(elastic, "test_index", []*config.Client{client}, nil, nil, nil)
	ctx := context.WithValue(context.Background(), "client", client.Name)

	query := "Max Mustermann"
	if _, err := r.Search(ctx, "all", query, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	funcScoreBytes, ok := capturedBody.Query["function_score"]
	if !ok {
		t.Fatal("expected function_score query in search request")
	}

	jsonStr := string(funcScoreBytes)

	// Verify single-level nested field [persons].role
	if !strings.Contains(jsonStr, `"path":"persons"`) {
		t.Errorf("expected JSON to contain persons nested path, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"fields":["persons.name"]`) {
		t.Errorf("expected JSON to contain persons.name search field, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"persons.role.keyword":{"value":"Autor"}`) {
		t.Errorf("expected JSON to contain Autor role filter, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"persons.role.keyword":{"value":"Übersetzer"}`) {
		t.Errorf("expected JSON to contain Übersetzer role filter, got: %s", jsonStr)
	}

	// Verify multi-level nested field [media.audio].type
	if !strings.Contains(jsonStr, `"path":"media.audio"`) {
		t.Errorf("expected JSON to contain media.audio nested path, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"fields":["media.audio.*"]`) {
		t.Errorf("expected JSON to contain media.audio.* search field, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"media.audio.type.keyword":{"value":"interview"}`) {
		t.Errorf("expected JSON to contain media.audio.type.keyword filter, got: %s", jsonStr)
	}

	// Verify root-level field category
	if !strings.Contains(jsonStr, `"category.keyword":{"value":"art"}`) {
		t.Errorf("expected JSON to contain category.keyword root term filter, got: %s", jsonStr)
	}

	// Verify zero-weight was excluded
	if strings.Contains(jsonStr, `"none"`) {
		t.Errorf("expected zero weight 'none' to be excluded from score functions, got: %s", jsonStr)
	}

	// Verify boost mode and score mode
	if !strings.Contains(jsonStr, `"boost_mode":"multiply"`) {
		t.Errorf("expected boost_mode multiply, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"score_mode":"max"`) {
		t.Errorf("expected score_mode max, got: %s", jsonStr)
	}
}

func TestElasticResolver_FieldWeightsPrecedence(t *testing.T) {
	var capturedBody struct {
		Query map[string]json.RawMessage `json:"query"`
	}

	elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatal(err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
				Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	globalWeights := map[string]map[string]float64{
		"[persons].role": {
			"author": 2.0,
		},
	}

	clientCustom := &config.Client{
		Name: "custom_client",
		FieldWeights: map[string]map[string]float64{
			"[persons].role": {
				"author": 5.0,
			},
		},
	}

	clientFallback := &config.Client{
		Name: "fallback_client",
	}

	r := NewElasticResolver(elastic, "test_index", []*config.Client{clientCustom, clientFallback}, nil, globalWeights, nil)

	// 1. Client with custom weights
	ctxCustom := context.WithValue(context.Background(), "client", clientCustom.Name)
	if _, err := r.Search(ctxCustom, "all", "test", nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	jsonCustom := string(capturedBody.Query["function_score"])
	if !strings.Contains(jsonCustom, `"weight":5`) {
		t.Errorf("expected client-specific weight 5, got: %s", jsonCustom)
	}

	// 2. Client with fallback to global weights
	ctxFallback := context.WithValue(context.Background(), "client", clientFallback.Name)
	if _, err := r.Search(ctxFallback, "all", "test", nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	jsonFallback := string(capturedBody.Query["function_score"])
	if !strings.Contains(jsonFallback, `"weight":2`) {
		t.Errorf("expected global fallback weight 2, got: %s", jsonFallback)
	}
}

func TestLoadRevCatConfig_SearchWeights(t *testing.T) {
	conf := &config.RevCatConfig{}
	if err := config.LoadRevCatConfig(config.ConfigFS, "revcat.toml", conf); err != nil {
		t.Fatalf("failed to load revcat.toml: %v", err)
	}

	if len(conf.ElasticSearch.SearchWeights) == 0 {
		t.Fatalf("expected searchweights to be parsed from revcat.toml, but got empty map")
	}

	expectedFields := map[string]float64{
		"title":            4.0,
		"persons.name":     4.0,
		"collectiontitle":  2.0,
		"series":           2.0,
		"tags":             2.0,
		"category":         1.5,
		"abstract":         1.1,
		"notes.title":      1.2,
		"notes.note":       1.0,
		"media.*.fulltext": 1.0,
	}

	for field, expectedWeight := range expectedFields {
		weight, ok := conf.ElasticSearch.SearchWeights[field]
		if !ok {
			t.Errorf("expected field %q to be present in searchweights", field)
			continue
		}
		if weight != expectedWeight {
			t.Errorf("expected field %q to have weight %v, got %v", field, expectedWeight, weight)
		}
	}

	// Verify client-specific searchweights
	var performanceClient, inkClient *config.Client
	for _, c := range conf.Client {
		if c.Name == "performance" {
			performanceClient = c
		}
		if c.Name == "ink" {
			inkClient = c
		}
	}

	if performanceClient == nil {
		t.Fatal("expected client 'performance' to exist in config")
	}
	if performanceClient.SearchWeights["title"] != 3.0 || performanceClient.SearchWeights["persons.name"] != 5.0 {
		t.Errorf("unexpected searchweights for client 'performance': %v", performanceClient.SearchWeights)
	}

	if inkClient == nil {
		t.Fatal("expected client 'ink' to exist in config")
	}
	if inkClient.SearchWeights["title"] != 3.0 || inkClient.SearchWeights["persons.name"] != 5.0 {
		t.Errorf("unexpected searchweights for client 'ink': %v", inkClient.SearchWeights)
	}
}

func TestElasticResolver_SearchWeightsPrecedence(t *testing.T) {
	var capturedBody struct {
		Query map[string]json.RawMessage `json:"query"`
	}

	elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatal(err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
				Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	globalSearchWeights := map[string]float64{
		"title":        7.0,
		"persons.name": 8.0,
	}

	clientCustom := &config.Client{
		Name: "custom_client",
		SearchWeights: map[string]float64{
			"title": 9.0,
		},
	}

	clientFallback := &config.Client{
		Name: "fallback_client",
	}

	r := NewElasticResolver(elastic, "test_index", []*config.Client{clientCustom, clientFallback}, globalSearchWeights, nil, nil)

	// 1. Client with custom weight for title (should be 9.0) and global weight for persons.name (should be 8.0)
	ctxCustom := context.WithValue(context.Background(), "client", clientCustom.Name)
	if _, err := r.Search(ctxCustom, "all", "test", nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	bodyBytes, _ := json.Marshal(capturedBody)
	jsonCustom := string(bodyBytes)
	if !strings.Contains(jsonCustom, `"title^9"`) {
		t.Errorf("expected client-specific search weight 'title^9', got: %s", jsonCustom)
	}
	if !strings.Contains(jsonCustom, `"persons.name^8"`) {
		t.Errorf("expected global search weight 'persons.name^8', got: %s", jsonCustom)
	}

	// 2. Client fallback: global weight for title (7.0) and fallback for tags (default 2.0)
	ctxFallback := context.WithValue(context.Background(), "client", clientFallback.Name)
	if _, err := r.Search(ctxFallback, "all", "test", nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	bodyBytesFallback, _ := json.Marshal(capturedBody)
	jsonFallback := string(bodyBytesFallback)
	if !strings.Contains(jsonFallback, `"title^7"`) {
		t.Errorf("expected global search weight 'title^7', got: %s", jsonFallback)
	}
	if !strings.Contains(jsonFallback, `"tags^2"`) {
		t.Errorf("expected default search weight 'tags^2', got: %s", jsonFallback)
	}
}

func TestElasticResolver_SearchWeightsFulltext(t *testing.T) {
	var capturedBody struct {
		Query map[string]json.RawMessage `json:"query"`
	}

	elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
				t.Fatal(err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
				Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := &config.Client{
		Name: "test_fulltext_client",
		SearchWeights: map[string]float64{
			"abstract":         3.5,
			"notes.title":      2.5,
			"notes.note":       1.8,
			"media.*.fulltext": 4.2,
		},
	}

	r := NewElasticResolver(elastic, "test_index", []*config.Client{client}, nil, nil, nil)
	ctx := context.WithValue(context.Background(), "client", client.Name)

	if _, err := r.Search(ctx, "fulltext", "test query", nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	bodyBytes, _ := json.Marshal(capturedBody)
	jsonStr := string(bodyBytes)

	if !strings.Contains(jsonStr, `"abstract^3.5"`) {
		t.Errorf("expected 'abstract^3.5' in fulltext search query, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"notes.title^2.5"`) {
		t.Errorf("expected 'notes.title^2.5' in fulltext search query, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"notes.note^1.8"`) {
		t.Errorf("expected 'notes.note^1.8' in fulltext search query, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"media.*.fulltext^4.2"`) {
		t.Errorf("expected 'media.*.fulltext^4.2' in fulltext search query, got: %s", jsonStr)
	}
}

func TestElasticResolver_EmptyQueryAcrossClients(t *testing.T) {
	var capturedBody struct {
		Query struct {
			Bool struct {
				Filter []map[string]json.RawMessage `json:"filter"`
				Must   []json.RawMessage            `json:"must"`
				Should []json.RawMessage            `json:"should"`
			} `json:"bool"`
		} `json:"query"`
	}
	var capturedSize int
	var capturedFrom int

	mockHitsJSON := `{
		"hits": {
			"total": {"value": 3, "relation": "eq"},
			"hits": [
				{
					"_id": "doc1",
					"_source": {
						"signature": "doc1",
						"source": "test",
						"category": ["zotero2!!PCB_Basel"],
						"acl": {
							"meta": ["global/guest"],
							"content": ["global/guest"]
						}
					}
				},
				{
					"_id": "doc2",
					"_source": {
						"signature": "doc2",
						"source": "test",
						"category": ["bangbang"],
						"acl": {
							"meta": ["global/guest"]
						}
					}
				},
				{
					"_id": "doc3_no_meta",
					"_source": {
						"signature": "doc3_no_meta",
						"source": "test",
						"category": ["zotero2!!PCB_Basel"],
						"acl": {
							"content": ["global/guest"]
						}
					}
				}
			]
		}
	}`

	elastic, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Transport: searchCaptureTransport(func(req *http.Request) (*http.Response, error) {
			capturedBody = struct {
				Query struct {
					Bool struct {
						Filter []map[string]json.RawMessage `json:"filter"`
						Must   []json.RawMessage            `json:"must"`
						Should []json.RawMessage            `json:"should"`
					} `json:"bool"`
				} `json:"query"`
			}{}
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(bodyBytes, &capturedBody); err != nil {
				t.Fatal(err)
			}
			var paging struct {
				From int `json:"from"`
				Size int `json:"size"`
			}
			_ = json.Unmarshal(bodyBytes, &paging)
			capturedFrom = paging.From
			capturedSize = paging.Size

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
				Body:       io.NopCloser(strings.NewReader(mockHitsJSON)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	clientPerformance := &config.Client{
		Name:   "performance",
		Groups: []string{"global/guest"},
		AND: []config.ClientANDQuery{
			{
				OR: []config.ClientOrQuery{
					{
						Field:  "category.keyword",
						Values: []string{"zotero2!!PCB_Basel", "bangbang"},
					},
				},
			},
		},
		FieldWeights: map[string]map[string]float64{
			"[persons].role": {"artist": 4.0},
		},
	}

	clientInk := &config.Client{
		Name:   "ink",
		Groups: []string{"global/guest"},
	}

	clientBaseline := &config.Client{
		Name: "default_baseline",
	}

	r := NewElasticResolver(elastic, "test_index", []*config.Client{clientPerformance, clientInk, clientBaseline}, nil, nil, nil)

	// 1. Performance client empty query
	t.Run("performance_client_empty_query", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "client", "performance")
		ctx = context.WithValue(ctx, "groups", []string{"global/guest"})
		size := 20
		first := 5
		res, err := r.Search(ctx, "all", "", nil, nil, nil, &first, &size, nil, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if res.TotalCount != 3 {
			t.Errorf("expected TotalCount 3, got %d", res.TotalCount)
		}
		// Doc 1 and Doc 2 have meta ACL, Doc 3 does not have meta ACL
		if len(res.Edges) != 2 {
			t.Errorf("expected 2 edges after ACL meta filtering, got %d", len(res.Edges))
		}
		if capturedSize != 20 {
			t.Errorf("expected size 20 in request, got %d", capturedSize)
		}
		if capturedFrom != 5 {
			t.Errorf("expected from 5 in request, got %d", capturedFrom)
		}
		if len(capturedBody.Query.Bool.Must) != 0 {
			t.Errorf("expected empty Must clauses for empty query, got %d", len(capturedBody.Query.Bool.Must))
		}
		if len(capturedBody.Query.Bool.Should) != 0 {
			t.Errorf("expected empty Should clauses for empty query, got %d", len(capturedBody.Query.Bool.Should))
		}
		// Check that category filter and ACL filter are present in Filter clauses
		filterBytes, _ := json.Marshal(capturedBody.Query.Bool.Filter)
		filterStr := string(filterBytes)
		if !strings.Contains(filterStr, "zotero2!!PCB_Basel") || !strings.Contains(filterStr, "bangbang") {
			t.Errorf("expected category filter values in Filter clause, got: %s", filterStr)
		}
		if !strings.Contains(filterStr, "acl.meta.keyword") || !strings.Contains(filterStr, "global/guest") {
			t.Errorf("expected ACL filter in Filter clause, got: %s", filterStr)
		}
	})

	// 2. Ink client empty query (no category filter)
	t.Run("ink_client_empty_query", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "client", "ink")
		ctx = context.WithValue(ctx, "groups", []string{"global/guest"})
		res, err := r.Search(ctx, "all", "", nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(res.Edges) != 2 {
			t.Errorf("expected 2 edges, got %d", len(res.Edges))
		}
		filterBytes, _ := json.Marshal(capturedBody.Query.Bool.Filter)
		filterStr := string(filterBytes)
		if strings.Contains(filterStr, "category.keyword") {
			t.Errorf("ink client should NOT have category.keyword filter, got: %s", filterStr)
		}
		if !strings.Contains(filterStr, "acl.meta.keyword") {
			t.Errorf("expected ACL filter in Filter clause, got: %s", filterStr)
		}
	})

	// 3. Baseline client empty query with default groups
	t.Run("baseline_client_empty_query", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "client", "default_baseline")
		ctx = context.WithValue(ctx, "groups", []string{"global/guest"})
		res, err := r.Search(ctx, "all", "", nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(res.Edges) != 2 {
			t.Errorf("expected 2 edges, got %d", len(res.Edges))
		}
	})
}
