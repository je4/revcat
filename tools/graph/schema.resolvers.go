package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"

	"github.com/je4/revcat/v2/tools/graph/model"
)

// ReferencesFull is the resolver for the referencesFull field.
func (r *mediathekFullEntryResolver) ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	return r.serverResolver.ReferencesFull(ctx, obj)
}

// Search is the resolver for the search field.
func (r *queryResolver) Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string) (*model.SearchResult, error) {
	return r.serverResolver.Search(ctx, query, facets, filter, vector, first, size, cursor)
}

// VectorSearch is the resolver for the vectorSearch field.
func (r *queryResolver) VectorSearch(ctx context.Context, filter []*model.InFilter, vector []float64, size int) (*model.SearchResult, error) {
	return r.serverResolver.VectorSearch(ctx, filter, vector, size)
}

// MediathekEntries is the resolver for the mediathekEntries field.
func (r *queryResolver) MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error) {
	return r.serverResolver.MediathekEntries(ctx, signatures)
}

// MediathekFullEntry returns MediathekFullEntryResolver implementation.
func (r *Resolver) MediathekFullEntry() MediathekFullEntryResolver {
	return &mediathekFullEntryResolver{r}
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mediathekFullEntryResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
