package graph

import (
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/utils/v2/pkg/zLogger"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

func NewResolver(elastic *elasticsearch.TypedClient, index string, logger zLogger.ZLogger) *Resolver {
	return &Resolver{
		elastic: elastic,
		index:   index,
		logger:  logger,
	}
}

type Resolver struct {
	elastic *elasticsearch.TypedClient
	logger  zLogger.ZLogger
	index   string
}
