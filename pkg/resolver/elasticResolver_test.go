package resolver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/je4/revcat/v2/config"
)

func TestLoadRevCatConfig_RoleWeights(t *testing.T) {
	conf := &config.RevCatConfig{}
	if err := config.LoadRevCatConfig(config.ConfigFS, "revcat.toml", conf); err != nil {
		t.Fatalf("failed to load revcat.toml: %v", err)
	}

	if len(conf.ElasticSearch.RoleWeights) == 0 {
		t.Fatalf("expected roleweights to be parsed from revcat.toml, but got empty map")
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
		weight, ok := conf.ElasticSearch.RoleWeights[role]
		if !ok {
			t.Errorf("expected role %q to be present in roleweights", role)
			continue
		}
		if weight != expectedWeight {
			t.Errorf("expected role %q to have weight %v, got %v", role, expectedWeight, weight)
		}
	}

	// Verify client-specific roleweights
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
	if performanceClient.RoleWeights["artist"] != 4.0 || performanceClient.RoleWeights["performer"] != 4.0 || performanceClient.RoleWeights["director"] != 1.5 {
		t.Errorf("unexpected roleweights for client 'performance': %v", performanceClient.RoleWeights)
	}

	if inkClient == nil {
		t.Fatal("expected client 'ink' to exist in config")
	}
	if inkClient.RoleWeights["author"] != 4.0 || inkClient.RoleWeights["editor"] != 2.0 {
		t.Errorf("unexpected roleweights for client 'ink': %v", inkClient.RoleWeights)
	}
}

func TestElasticResolver_FunctionScoreQueryGeneration(t *testing.T) {
	roleWeights := map[string]float64{
		"Autor":      3.0,
		"Übersetzer": 1.1,
	}

	r := &ElasticResolver{
		roleWeights: roleWeights,
	}

	query := "Max Mustermann"

	// Mock building query matching ElasticResolver logic
	roles := []string{"Autor", "Übersetzer"}
	var scoreFunctions []types.FunctionScore
	for _, role := range roles {
		weight := r.roleWeights[role]
		w := types.Float64(weight)
		rRole := role
		scoreFunctions = append(scoreFunctions, types.FunctionScore{
			Filter: &types.Query{
				Nested: &types.NestedQuery{
					Path: "persons",
					Query: types.Query{
						Bool: &types.BoolQuery{
							Must: []types.Query{
								{
									SimpleQueryString: &types.SimpleQueryStringQuery{
										Query:  query,
										Fields: []string{"persons.name"},
									},
								},
								{
									Term: map[string]types.TermQuery{
										"persons.role.keyword": {
											Value: rRole,
										},
									},
								},
							},
						},
					},
				},
			},
			Weight: &w,
		})
	}

	if len(scoreFunctions) != 2 {
		t.Fatalf("expected 2 score functions, got %d", len(scoreFunctions))
	}

	data, err := json.Marshal(scoreFunctions)
	if err != nil {
		t.Fatalf("failed to marshal score functions: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"path":"persons"`) {
		t.Errorf("expected JSON to contain persons nested path, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"persons.role.keyword":{"value":"Autor"}`) {
		t.Errorf("expected JSON to contain Autor role filter, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"persons.role.keyword":{"value":"Übersetzer"}`) {
		t.Errorf("expected JSON to contain Übersetzer role filter, got: %s", jsonStr)
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
		weight, ok := conf.ElasticSearch.FieldWeights[field]
		if !ok {
			t.Errorf("expected field %q to be present in fieldweights", field)
			continue
		}
		if weight != expectedWeight {
			t.Errorf("expected field %q to have weight %v, got %v", field, expectedWeight, weight)
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
	if performanceClient.FieldWeights["title"] != 4.0 {
		t.Errorf("unexpected fieldweights for client 'performance': %v", performanceClient.FieldWeights)
	}

	if inkClient == nil {
		t.Fatal("expected client 'ink' to exist in config")
	}
	if inkClient.FieldWeights["title"] != 5.0 {
		t.Errorf("unexpected fieldweights for client 'ink': %v", inkClient.FieldWeights)
	}
}

func TestElasticResolver_FieldWeightsResolution(t *testing.T) {
	// 1. Defaults when no config is provided
	rDefault := &ElasticResolver{}
	getWeightDefault := func(name string, client *config.Client) float64 {
		if client != nil && client.FieldWeights != nil {
			if w, ok := client.FieldWeights[name]; ok && w > 0 {
				return w
			}
		}
		if rDefault.fieldWeights != nil {
			if w, ok := rDefault.fieldWeights[name]; ok && w > 0 {
				return w
			}
		}
		if def, ok := defaultFieldWeights[name]; ok {
			return def
		}
		return 1.0
	}

	if getWeightDefault("title", nil) != 4.0 {
		t.Errorf("expected default title weight 4.0, got %v", getWeightDefault("title", nil))
	}
	if getWeightDefault("category", nil) != 1.5 {
		t.Errorf("expected default category weight 1.5, got %v", getWeightDefault("category", nil))
	}

	// 2. Custom global weights override defaults
	rCustom := &ElasticResolver{
		fieldWeights: map[string]float64{
			"title": 6.0,
		},
	}
	getWeightCustom := func(name string, client *config.Client) float64 {
		if client != nil && client.FieldWeights != nil {
			if w, ok := client.FieldWeights[name]; ok && w > 0 {
				return w
			}
		}
		if rCustom.fieldWeights != nil {
			if w, ok := rCustom.fieldWeights[name]; ok && w > 0 {
				return w
			}
		}
		if def, ok := defaultFieldWeights[name]; ok {
			return def
		}
		return 1.0
	}

	if getWeightCustom("title", nil) != 6.0 {
		t.Errorf("expected global custom title weight 6.0, got %v", getWeightCustom("title", nil))
	}
	if getWeightCustom("abstract", nil) != 1.1 {
		t.Errorf("expected fallback default abstract weight 1.1, got %v", getWeightCustom("abstract", nil))
	}

	// 3. Client-specific weights override global weights
	client := &config.Client{
		FieldWeights: map[string]float64{
			"title": 8.0,
			"tags":  5.0,
		},
	}

	if getWeightCustom("title", client) != 8.0 {
		t.Errorf("expected client-specific title weight 8.0, got %v", getWeightCustom("title", client))
	}
	if getWeightCustom("tags", client) != 5.0 {
		t.Errorf("expected client-specific tags weight 5.0, got %v", getWeightCustom("tags", client))
	}
	if getWeightCustom("abstract", client) != 1.1 {
		t.Errorf("expected fallback abstract weight 1.1, got %v", getWeightCustom("abstract", client))
	}
}
