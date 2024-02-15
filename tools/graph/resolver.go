package graph

import (
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/utils/v2/pkg/zLogger"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

func NewResolver(serverResolver resolver.Resolver, logger zLogger.ZLogger) *Resolver {
	r := &Resolver{
		logger:         logger,
		serverResolver: serverResolver,
	}
	return r
}

type Resolver struct {
	serverResolver resolver.Resolver
	//	elastic     *elasticsearch.TypedClient
	logger zLogger.ZLogger
	//	index       string
	//	objectCache gcache.Cache
	//	client      map[string]*config.Client
}
