package graph

import (
	"context"
	emperror "emperror.dev/errors"
	"encoding/json"
	"github.com/bluele/gcache"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
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
	if len(newSignatures) > 0 {
		mgetResponse, err := r.elastic.Mget().Index(r.index).Ids(newSignatures...).SourceExcludes_("title_vector", "content_vector").Do(ctx)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot load '%s' entries %v", r.index, signatures)
		}
		for _, docInt := range mgetResponse.Docs {
			doc, ok := docInt.(*types.GetResult)
			if !ok {
				return nil, emperror.Errorf("cannot convert doc %v to map", docInt)
			}
			jsonBytes := doc.Source_
			source := sourcetype.SourceData{ID: doc.Id_}
			if err := json.Unmarshal(jsonBytes, &source); err != nil {
				return nil, emperror.Wrapf(err, "cannot unmarshal source %v", source)
			}
			result = append(result, source)
			r.objectCache.Set(doc.Id_, source)
		}
	}
	return result, nil
}
