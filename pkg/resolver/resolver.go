package resolver

import (
	"context"
	"github.com/je4/revcat/v2/tools/graph/model"
)

type Resolver interface {
	// Search is the resolver for the search field.
	Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string, sort []*model.SortField) (*model.SearchResult, error)

	// MediathekEntries is the resolver for the mediathekEntries field.
	MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error)

	ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error)
}
