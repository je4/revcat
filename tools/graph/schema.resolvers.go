package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	emperrors "emperror.dev/errors"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

// ReferencesFull is the resolver for the referencesFull field.
func (r *mediathekFullEntryResolver) ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, emperrors.Errorf("error: %s", errValue)
	}
	var result = make([]*model.MediathekBaseEntry, 0)
	var signatures = []string{}
	for _, ref := range obj.Base.References {
		signatures = append(signatures, ref.Signature)
	}
	if len(signatures) == 0 {
		return result, nil
	}
	docs, err := r.loadEntries(ctx, signatures)
	if err != nil {
		return nil, emperrors.Wrapf(err, "cannot load entries %v", signatures)
	}
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot get groups from context")
	}
	var access = make(map[string]bool)
	for _, doc := range docs {
		for t, acls := range doc.ACL {
			for _, group := range groups {
				if slices.Contains(acls, group) {
					access[strings.ToLower(t)] = true
					break
				}
			}
		}
		if ok, found := access["meta"]; ok && found {
			entry := sourceToMediathekBaseEntry(&doc)
			result = append(result, entry)
		}
	}
	return result, nil
}

// Search is the resolver for the search field.
func (r *queryResolver) Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, first *int, after *string, last *int, before *string) (*model.SearchResult, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, emperrors.Errorf("error: %s", errValue)
	}
	var from = 0
	var size = 25

	if first != nil && last != nil {
		if *first > *last {
			return nil, emperrors.Errorf("first cannot be greater than last")
		}
		from = *first
		size = *last - *first
		if size == 0 {
			size = 25
		}
	}
	if after != nil && before != nil && *after != "" && *before != "" {
		return nil, emperrors.Errorf("after and before cannot be used together")
	}
	if after != nil {
		if *after != "" {
			crs := &cursor{}
			afterJson, err := base64.StdEncoding.DecodeString(*after)
			if err != nil {
				return nil, emperrors.Wrapf(err, "cannot decode after cursor '%s'", *after)
			}
			if err := json.Unmarshal(afterJson, crs); err != nil {
				return nil, emperrors.Wrapf(err, "cannot unmarshal after cursor '%s'", afterJson)
			}
			from = crs.From + 1
			size = crs.Size
		}
	}
	if before != nil {
		if *before != "" {
			crs := &cursor{}
			beforeJson, err := base64.StdEncoding.DecodeString(*before)
			if err != nil {
				return nil, emperrors.Wrapf(err, "cannot decode before cursor '%s'", *before)
			}
			if err := json.Unmarshal(beforeJson, crs); err != nil {
				return nil, emperrors.Wrapf(err, "cannot unmarshal before cursor '%s'", beforeJson)
			}
			from = crs.From - size
			if from < 0 {
				from = 0
			}
			size = crs.Size
		}
	}
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot get groups from context")
	}
	clientName, err := stringFromContext(ctx, "client")
	if err != nil || clientName == "" {
		return nil, emperrors.Wrap(err, "cannot get client from context")
	}
	client, ok := r.client[clientName]
	if !ok {
		return nil, emperrors.Errorf("client '%s' not found", clientName)
	}

	baseQuery := types.BoolQuery{
		Must:               []types.Query{},
		Should:             []types.Query{},
		MinimumShouldMatch: 1,
	}
	for _, q := range client.AND {
		if q.Field == "" {
			continue
		}
		for _, val := range q.Values {
			baseQuery.Must = append(baseQuery.Must, types.Query{
				Term: map[string]types.TermQuery{
					q.Field: {
						Value: val,
					},
				},
			})
			// baseQuery.Must = append(baseQuery.Must, createFilterQuery(q.Field, val))
		}
	}
	for _, q := range client.OR {
		if q.Field == "" {
			continue
		}
		for _, val := range q.Values {
			baseQuery.Should = append(baseQuery.Should, types.Query{
				Term: map[string]types.TermQuery{
					q.Field: {
						Value: val,
					},
				},
			})
			//			baseQuery.Should = append(baseQuery.Should, createFilterQuery(q.Field, val))
		}
	}
	if len(baseQuery.Should) == 0 {
		baseQuery.MinimumShouldMatch = 0
	}
	aclQuery := types.BoolQuery{
		Must:               []types.Query{},
		Should:             []types.Query{},
		MinimumShouldMatch: 1,
	}
	grps := []string{}
	for _, grp := range client.Groups {
		grps = append(grps, strings.ToLower(grp))
	}
	for _, grp := range groups {
		grps = append(grps, strings.ToLower(grp))
	}
	slices.Sort(grps)
	grps = slices.Compact(grps)
	for _, grp := range grps {
		aclQuery.Must = append(aclQuery.Must, types.Query{
			Term: map[string]types.TermQuery{
				"acl.meta.keyword": {
					Value: grp,
				},
			},
		})
		//		aclQuery.Must = append(aclQuery.Must, createFilterQuery("acl.meta", grp))
	}
	if len(aclQuery.Should) == 0 {
		aclQuery.MinimumShouldMatch = 0
	}

	//
	// start building query
	//

	var esFilter = []types.Query{
		types.Query{Bool: &baseQuery},
		types.Query{Bool: &aclQuery},
	}
	var esPostFilter = []types.Query{
		types.Query{Bool: &baseQuery},
		types.Query{Bool: &aclQuery},
	}
	var esAggs = map[string]types.Aggregations{}

	for _, f := range filter {
		newFilter, err := createFilterQuery(f)
		if err != nil {
			return nil, emperrors.Wrapf(err, "cannot create filter query for %v", f)
		}
		esFilter = append(esFilter, newFilter)
	}

	for _, f := range facets {
		if err != nil {
			return nil, emperrors.Wrapf(err, "cannot create filter query for %v", f)
		}
		newFilter, err := createFilterQuery(f.Query)
		if err != nil {
			return nil, emperrors.Wrapf(err, "cannot create facet filter query for %v", f)
		}

		esPostFilter = append(esPostFilter, newFilter)
		if f.Term != nil {
			termAgg := &types.TermsAggregation{
				Field: &f.Term.Field,
				//			Name:  &f.Name,
			}
			if len(f.Term.Include) > 0 {
				termAgg.Include = f.Term.Include
				s := len(f.Term.Include)
				termAgg.Size = &s
			}
			esAggs[f.Term.Name] = types.Aggregations{
				Terms: termAgg,
			}
		}
	}

	esMust := []types.Query{}
	if query != "" {
		esMust = append(esMust, types.Query{
			SimpleQueryString: &types.SimpleQueryStringQuery{
				Query: query,
			},
		})
	}

	if len(facets) > 0 {

	}

	resp, err := r.elastic.Search().
		Index(r.index).
		Request(&search.Request{
			Query: &types.Query{
				Bool: &types.BoolQuery{
					Filter: esFilter,
					Must:   esMust,
				},
			},
			Aggregations: esAggs,
			PostFilter: &types.Query{
				Bool: &types.BoolQuery{
					Filter: esPostFilter,
				},
			},
		}).
		From(from).
		Size(size).
		Do(ctx)
	if err != nil {
		return nil, emperrors.Wrapf(err, "cannot search for '%s'", query)
	}
	var result = &model.SearchResult{
		TotalCount: int(resp.Hits.Total.Value),
		Edges:      make([]*model.MediathekFullEntry, 0),
		Facets:     make([]*model.Facet, 0),
		PageInfo:   &model.PageInfo{},
	}
	for name, bucketAny := range resp.Aggregations {
		facet := &model.Facet{
			Field:  name,
			Values: make([]*model.FacetValue, 0),
		}
		switch bucket := bucketAny.(type) {
		case *types.StringTermsAggregate:
			switch bucketType1 := bucket.Buckets.(type) {
			case []types.StringTermsBucket:
				for _, stb := range bucketType1 {
					switch kt := stb.Key.(type) {
					case string:
						facet.Values = append(facet.Values, &model.FacetValue{
							ValStr: &kt,
							Count:  int(stb.DocCount),
						})
					case int64:
						intVal := int(kt)
						facet.Values = append(facet.Values, &model.FacetValue{
							ValInt: &intVal,
							Count:  int(stb.DocCount),
						})
					default:
						return nil, emperrors.Errorf("unknown bucket key type of StringTermsBucket key %T", kt)
					}
				}
				//			case map[string]any:
			default:
				return nil, emperrors.Errorf("unknown bucket type of StringTermsAggregate %T", bucketType1)
			}
		default:
			return nil, emperrors.Errorf("unknown bucket type %T", bucket)
		}
		result.Facets = append(result.Facets, facet)
	}
	if result.TotalCount > from+size {
		result.PageInfo.HasNextPage = true

		cFrom := from + size - 1
		if cFrom >= result.TotalCount {
			cFrom = result.TotalCount - 1
		}
		endCursor, err := json.Marshal(cursor{
			From: cFrom,
			Size: size,
		})
		if err != nil {
			return nil, emperrors.Wrap(err, "cannot marshal end cursor")
		}
		result.PageInfo.EndCursor = base64.StdEncoding.EncodeToString(endCursor)
	}
	if from > 0 {
		result.PageInfo.HasPreviousPage = true
		startCursor, err := json.Marshal(cursor{
			From: from,
			Size: size,
		})
		if err != nil {
			return nil, emperrors.Wrap(err, "cannot marshal start cursor")
		}
		result.PageInfo.StartCursor = url.QueryEscape(base64.StdEncoding.EncodeToString(startCursor))
	}
	for _, hit := range resp.Hits.Hits {
		source := &sourcetype.SourceData{}
		if err := json.Unmarshal(hit.Source_, source); err != nil {
			return nil, emperrors.Wrapf(err, "cannot unmarshal hit %v", hit)
		}
		entry := sourceToMediathekFullEntry(source)
		result.Edges = append(result.Edges, entry)
	}
	return result, nil
}

// MediathekEntries is the resolver for the mediathekEntries field.
func (r *queryResolver) MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, emperrors.Errorf("error: %s", errValue)
	}
	docs, err := r.loadEntries(ctx, signatures)
	if err != nil {
		return nil, emperrors.Wrapf(err, "cannot load entries %v", signatures)
	}

	entries := make([]*model.MediathekFullEntry, 0)
	var access = make(map[string]bool)
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot get groups from context")
	}
	for _, doc := range docs {
		for t, acls := range doc.ACL {
			for _, group := range groups {
				if slices.Contains(acls, group) {
					access[strings.ToLower(t)] = true
					break
				}
			}
		}
		if ok, found := access["meta"]; ok && found {
			entry := sourceToMediathekFullEntry(&doc)
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// MediathekFullEntry returns MediathekFullEntryResolver implementation.
func (r *Resolver) MediathekFullEntry() MediathekFullEntryResolver {
	return &mediathekFullEntryResolver{r}
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mediathekFullEntryResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
type cursor struct {
	From int `json:"from"`
	Size int `json:"size"`
}
