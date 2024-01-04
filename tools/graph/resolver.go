package graph

import (
	"context"
	emperror "emperror.dev/errors"
	"encoding/json"
	"github.com/bluele/gcache"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/utils/v2/pkg/zLogger"
	"time"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

func NewResolver(elastic *elasticsearch.TypedClient, index string, clients []*config.Client, logger zLogger.ZLogger) *Resolver {
	r := &Resolver{
		elastic:     elastic,
		index:       index,
		logger:      logger,
		objectCache: gcache.New(800).LRU().Expiration(time.Minute).Build(),
		client:      make(map[string]*config.Client),
	}
	for _, client := range clients {
		r.client[client.Name] = client
	}
	return r
}

type Resolver struct {
	elastic     *elasticsearch.TypedClient
	logger      zLogger.ZLogger
	index       string
	objectCache gcache.Cache
	client      map[string]*config.Client
}

func (r *Resolver) loadEntries(ctx context.Context, signatures []string) ([]sourcetype.SourceData, error) {
	var result = make([]sourcetype.SourceData, 0)
	var newSignatures = make([]string, 0)
	for _, signature := range signatures {
		if obj, err := r.objectCache.Get(signature); err == nil {
			if source, ok := obj.(sourcetype.SourceData); ok {
				result = append(result, source)
				continue
			}
		}
		newSignatures = append(newSignatures, signature)
	}
	mgetResponse, err := r.elastic.Mget().Index(r.index).Ids(newSignatures...).SourceExcludes_("title_vector", "content_vector").Do(ctx)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot load '%s' entries %v", r.index, signatures)
	}
	for _, docInt := range mgetResponse.Docs {
		doc, ok := docInt.(map[string]interface{})
		if !ok {
			return nil, emperror.Errorf("cannot convert doc %v to map", docInt)
		}
		if found, ok := doc["found"].(bool); !ok || !found {
			return nil, emperror.Errorf("document %s not found", doc["id"])
		}
		id, ok := doc["_id"].(string)
		if !ok {
			return nil, emperror.Errorf("cannot convert doc id %v to string", doc["_id"])
		}
		sourceMap, ok := doc["_source"].(map[string]interface{})
		if !ok {
			return nil, emperror.Errorf("cannot convert doc source %v to map", doc["_source"])
		}
		jsonBytes, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot marshal source %v", sourceMap)
		}
		source := sourcetype.SourceData{ID: id}
		if err := json.Unmarshal(jsonBytes, &source); err != nil {
			return nil, emperror.Wrapf(err, "cannot unmarshal source %v", source)
		}
		result = append(result, source)
		r.objectCache.Set(id, source)
	}
	return result, nil
}
