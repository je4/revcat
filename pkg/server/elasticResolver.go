package server

import (
	"context"
	emperrors "emperror.dev/errors"
	"encoding/json"
	"github.com/bluele/gcache"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
	"github.com/je4/utils/v2/pkg/zLogger"
	"slices"
	"strings"
)

func NewElasticResolver(elastic *elasticsearch.TypedClient, index string, client map[string]*config.Client, logger zLogger.ZLogger) *ElasticResolver {
	return &ElasticResolver{
		elastic:     elastic,
		index:       index,
		logger:      logger,
		objectCache: gcache.New(800).LRU().Build(),
		client:      client,
	}
}

type ElasticResolver struct {
	elastic     *elasticsearch.TypedClient
	logger      zLogger.ZLogger
	index       string
	objectCache gcache.Cache
	client      map[string]*config.Client
}

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *ElasticResolver) buildBaseFilter(clientName string, groups []string) ([]types.Query, error) {
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

	var esFilter = []types.Query{
		types.Query{Bool: &baseQuery},
		types.Query{Bool: &aclQuery},
	}

	return esFilter, nil
}

// Search is the resolver for the search field.
func (r *ElasticResolver) Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string) (*model.SearchResult, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, emperrors.Errorf("%s", errValue)
	}
	var from = 0
	var num = 25

	if first != nil && size != nil {
		from = *first
		num = *size
	}
	if cursor != nil && *cursor != "" {
		crs, err := DecodeCursor(*cursor)
		if err != nil {
			return nil, emperrors.Wrapf(err, "cannot decode cursor '%s'", *cursor)
		}
		from = crs.From + 1
		num = crs.Size
	}

	if from < 0 {
		from = 0
	}
	if num <= 0 {
		num = 25
	}
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot get groups from context")
	}
	clientName, err := stringFromContext(ctx, "client")
	if err != nil || clientName == "" {
		return nil, emperrors.Wrap(err, "cannot get client from context")
	}

	esFilter, err := r.buildBaseFilter(clientName, groups)
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot build base filter")
	}
	var esPostFilter = []types.Query{
		//		types.Query{Bool: &baseQuery},
		//		types.Query{Bool: &aclQuery},
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
		newFilter, err := createFilterQuery(f.Query)
		if err != nil {
			return nil, emperrors.Wrapf(err, "cannot create facet filter query for %v", f)
		}

		esPostFilter = append(esPostFilter, newFilter)
	}
	for _, f := range facets {
		facetFilter := []types.Query{}
		for _, f2 := range facets {
			if f2.Term.Name == f.Term.Name {
				continue
			}
			newFilter, err := createFilterQuery(f2.Query)
			if err != nil {
				return nil, emperrors.Wrapf(err, "cannot create facet filter query for %v", f2)
			}

			facetFilter = append(facetFilter, newFilter)
		}
		if f.Term != nil {
			termAgg := &types.TermsAggregation{
				Field:       &f.Term.Field,
				Size:        &f.Term.Size,
				MinDocCount: &f.Term.MinDocCount,
				//			Name:  &f.Name,
			}
			if len(f.Term.Include) == 1 {
				termAgg.Include = f.Term.Include[0]
			} else {
				if len(f.Term.Include) > 1 {
					termAgg.Include = f.Term.Include
					s := len(f.Term.Include)
					termAgg.Size = &s
					zero := 0
					termAgg.MinDocCount = &zero
				}
			}

			esAggs[f.Term.Name] = types.Aggregations{
				Filter: &types.Query{
					Bool: &types.BoolQuery{
						Filter: facetFilter,
					},
				},
				Aggregations: map[string]types.Aggregations{
					"theAggregation": types.Aggregations{
						Terms: termAgg,
					},
				},
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

	searchRequest := &search.Request{
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
	}

	resp, err := r.elastic.Search().
		Index(r.index).
		SourceExcludes_("title_vector", "content_vector").
		Request(searchRequest).
		From(from).
		Size(num).
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
			Name:   name,
			Values: make([]model.FacetValue, 0),
		}
		filterAgg, ok := bucketAny.(*types.FilterAggregate)
		if !ok {
			return nil, emperrors.Errorf("unknown base bucket type %T in %s", bucketAny, name)
		}
		theAgg, ok := filterAgg.Aggregations["theAggregation"]
		if !ok {
			return nil, emperrors.Errorf("theAggregation not found in filter aggregate %s", name)
		}
		switch bucket := theAgg.(type) {
		case *types.StringTermsAggregate:
			switch bucketType1 := bucket.Buckets.(type) {
			case []types.StringTermsBucket:
				for _, stb := range bucketType1 {
					switch kt := stb.Key.(type) {
					case string:
						facet.Values = append(facet.Values, &model.FacetValueString{
							StrVal: kt,
							Count:  int(stb.DocCount),
						})
					case int64:
						intVal := int(kt)
						facet.Values = append(facet.Values, &model.FacetValueInt{
							IntVal: intVal,
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
	r.logger.Debug().Msgf("total count %d, from %d, num %d", result.TotalCount, from, num)
	if result.TotalCount > from+num {
		result.PageInfo.HasNextPage = true
		nFrom := min(from+num-1, result.TotalCount-1)
		r.logger.Debug().Msgf("next: from %d, num %d", nFrom, num)
		if result.PageInfo.EndCursor, err = NewCursor(nFrom, num).Encode(); err != nil {
			return nil, emperrors.Wrap(err, "cannot marshal end cursor")
		}
	}
	if from > 0 {
		result.PageInfo.HasPreviousPage = true
		nFrom := max(from-num-1, -1)
		r.logger.Debug().Msgf("prev: from %d, num %d", nFrom, num)
		if result.PageInfo.StartCursor, err = NewCursor(nFrom, num).Encode(); err != nil {
			return nil, emperrors.Wrap(err, "cannot marshal end cursor")
		}
	}
	result.PageInfo.CurrentCursor, err = NewCursor(from, num).Encode()
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot marshal current cursor")
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
