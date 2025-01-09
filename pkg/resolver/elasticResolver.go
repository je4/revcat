package resolver

import (
	"context"
	"emperror.dev/errors"
	"encoding/json"
	"github.com/bluele/gcache"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
	"github.com/je4/utils/v2/pkg/zLogger"
	"regexp"
	"slices"
	"strings"
	"time"
)

func NewElasticResolver(elastic *elasticsearch.TypedClient, index string, clients []*config.Client, logger zLogger.ZLogger) *ElasticResolver {
	r := &ElasticResolver{
		elastic:     elastic,
		index:       index,
		logger:      logger,
		objectCache: gcache.New(800).LRU().Build(),
		client:      make(map[string]*config.Client),
	}
	for _, client := range clients {
		r.client[client.Name] = client
	}
	return r
}

type ElasticResolver struct {
	elastic     *elasticsearch.TypedClient
	logger      zLogger.ZLogger
	index       string
	objectCache gcache.Cache
	client      map[string]*config.Client
	jwtKey      string
	jwtAlgs     []string
	jwtMaxAge   time.Duration
}

func BuildBaseFilter(client *config.Client, groups ...string) ([]types.Query, error) {
	/*
		client, ok := r.client[clientName]
		if !ok {
			return nil, errors.Errorf("client '%s' not found", clientName)
		}
	*/
	baseQuery := types.BoolQuery{
		Must: []types.Query{},
	}
	for _, and := range client.AND {
		andQuery := types.Query{
			Bool: &types.BoolQuery{
				MinimumShouldMatch: 1,
				Should:             []types.Query{},
			},
		}
		for _, q := range and.OR {
			if q.Field == "" {
				continue
			}
			andQuery.Bool.Should = append(andQuery.Bool.Should, types.Query{
				Terms: &types.TermsQuery{
					TermsQuery: map[string]types.TermsQueryField{
						q.Field: q.Values,
					},
				},
			})
		}
		baseQuery.Must = append(baseQuery.Must, andQuery)

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
		aclQuery.Should = append(aclQuery.Should, types.Query{
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

func (r *ElasticResolver) loadEntries(ctx context.Context, signatures []string) ([]sourcetype.SourceData, error) {
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
			return nil, errors.Wrapf(err, "cannot load '%s' entries %v", r.index, signatures)
		}
		for _, docInt := range mgetResponse.Docs {
			doc, ok := docInt.(*types.GetResult)
			if !ok {
				return nil, errors.Errorf("cannot convert doc %v to map", docInt)
			}
			if doc.Source_ == nil {
				return nil, errors.Errorf("source of doc %v is nil", doc)
			}
			jsonBytes := doc.Source_
			source := sourcetype.SourceData{ID: doc.Id_}
			if err := json.Unmarshal(jsonBytes, &source); err != nil {
				return nil, errors.Wrapf(err, "cannot unmarshal source %v", source)
			}
			result = append(result, source)
			r.objectCache.Set(doc.Id_, source)
		}
	}
	return result, nil
}

var sortFieldRegexp = regexp.MustCompile(`^[a-zA-Z0-9_.]*$`)

// Search is the resolver for the search field.
func (r *ElasticResolver) Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string, sort []*model.SortField) (*model.SearchResult, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, errors.Errorf("%s", errValue)
	}
	var from = 0
	var num = 25

	if first != nil {
		from = *first
	}
	if size != nil {
		num = *size
	}
	if cursor != nil && *cursor != "" {
		crs, err := DecodeCursor(*cursor)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decode cursor '%s'", *cursor)
		}
		from = crs.From
		num = crs.Size
	}

	if from < 0 {
		from = 0
	}
	if num < 0 {
		num = 25
	}
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, errors.Wrap(err, "cannot get groups from context")
	}
	clientName, err := stringFromContext(ctx, "client")
	if err != nil || clientName == "" {
		return nil, errors.Wrap(err, "cannot get client from context")
	}

	client, ok := r.client[clientName]
	if !ok {
		return nil, errors.Errorf("client '%s' not found", clientName)
	}
	esFilter, err := BuildBaseFilter(client, groups...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build base filter")
	}
	var esPostFilter = []*types.Query{
		//		types.Query{Bool: &baseQuery},
		//		types.Query{Bool: &aclQuery},
	}
	var esAggs = map[string]types.Aggregations{}

	for _, f := range filter {
		newFilter, err := createFilterQuery(f)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create filter query for %v", f)
		}
		esFilter = append(esFilter, *newFilter)
	}

	for _, f := range facets {
		newFilter, err := createFilterQuery(f.Query)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create facet filter query for %v", f)
		}
		if newFilter != nil {
			if f.Query.BoolTerm != nil && f.Query.BoolTerm.And {
				esFilter = append(esFilter, *newFilter)
			} else if f.Query.ExistsTerm != nil {
				esFilter = append(esFilter, *newFilter)
			} else {
				esPostFilter = append(esPostFilter, newFilter)
			}
		}
	}
	for _, f := range facets {
		facetFilter := []*types.Query{}
		//aggInclude := []string{}
		for _, f2 := range facets {
			if f2.Term.Name == f.Term.Name {
				continue
			}
			newFilter, err := createFilterQuery(f2.Query)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot create facet filter query for %v", f2)
			}
			if newFilter != nil {
				facetFilter = append(facetFilter, newFilter)
			}
		}
		if f.Term != nil {
			termAgg := &types.TermsAggregation{
				Field:       &f.Term.Field,
				Size:        &f.Term.Size,
				MinDocCount: &f.Term.MinDocCount,
				//				Exclude:     []string{"bangbang!!.*"},
				//Include: aggInclude,
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

			agg := types.Aggregations{
				Aggregations: map[string]types.Aggregations{
					"theAggregation": types.Aggregations{
						Terms: termAgg,
					},
				},
			}
			if len(facetFilter) > 0 {
				agg.Filter = &types.Query{
					Bool: &types.BoolQuery{
						Filter: []types.Query{},
					},
				}
				for _, ff := range facetFilter {
					if ff != nil {
						agg.Filter.Bool.Filter = append(agg.Filter.Bool.Filter, *ff)
					}
				}
			} else {
				agg.Filter = &types.Query{
					MatchAll: types.NewMatchAllQuery(),
				}
			}
			esAggs[f.Term.Name] = agg
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
	if len(vector) > 0 {
		vectorBytes, err := json.Marshal(vector)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal params")
		}
		scriptSource := "cosineSimilarity(params.queryVector, 'content_vector')"
		esMust = append(esMust, types.Query{
			ScriptScore: &types.ScriptScoreQuery{
				Query: &types.Query{
					Exists: &types.ExistsQuery{
						Field: "content_vector",
					},
				},
				Script: types.Script{
					Source: &scriptSource,
					Params: map[string]json.RawMessage{
						"queryVector": vectorBytes,
					},
				},
			},
		})
	}

	if len(facets) > 0 {

	}

	searchRequest := &search.Request{}
	if len(esAggs) > 0 {
		searchRequest.Aggregations = esAggs
	}
	if len(esPostFilter) > 0 {
		searchRequest.PostFilter = &types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{},
			},
		}
		for _, f := range esPostFilter {
			if f != nil {
				searchRequest.PostFilter.Bool.Filter = append(searchRequest.PostFilter.Bool.Filter, *f)
			}
		}
	}
	if searchRequest.Query == nil {
		searchRequest.Query = &types.Query{
			Bool: &types.BoolQuery{
				Filter: esFilter, // []types.Query{},
				Must:   esMust,
			},
		}
	}
	sorts := []types.SortCombinations{}
	for _, s := range sort {
		if !sortFieldRegexp.MatchString(s.Field) {
			return nil, errors.Errorf("invalid sort field '%s'", s.Field)
		}
		var order sortorder.SortOrder
		switch strings.ToLower(s.Order) {
		case "asc":
			order = sortorder.Asc
		case "desc":
			order = sortorder.Desc
		default:
			order = sortorder.Asc
		}
		sort := types.SortOptions{SortOptions: map[string]types.FieldSort{
			s.Field: {Order: &order},
		}}
		sorts = append(sorts, sort)
	}

	if searchRequest.Query != nil {
		boostQuery := &types.Query{
			Boosting: &types.BoostingQuery{
				Positive: searchRequest.Query,
				Negative: &types.Query{
					Bool: &types.BoolQuery{
						MustNot: []types.Query{
							types.Query{
								Exists: &types.ExistsQuery{
									Field: "poster",
								},
							},
						},
					},
				},
				NegativeBoost: 0.85,
			},
		}
		searchRequest.Query = boostQuery
	}

	elasticQuery := r.elastic.Search().
		Index(r.index).
		SourceExcludes_("title_vector", "content_vector").
		Request(searchRequest).
		From(from).
		Size(num)

	if len(sorts) > 0 {
		elasticQuery = elasticQuery.Sort(sorts...)
	}
	resp, err := elasticQuery.Do(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot search for '%s'", query)
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
			return nil, errors.Errorf("unknown base bucket type %T in %s", bucketAny, name)
		}
		theAgg, ok := filterAgg.Aggregations["theAggregation"]
		if !ok {
			return nil, errors.Errorf("theAggregation not found in filter aggregate %s", name)
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
						return nil, errors.Errorf("unknown bucket key type of StringTermsBucket key %T", kt)
					}
				}
				//			case map[string]any:
			default:
				return nil, errors.Errorf("unknown bucket type of StringTermsAggregate %T", bucketType1)
			}
		default:
			return nil, errors.Errorf("unknown bucket type %T", bucket)
		}
		result.Facets = append(result.Facets, facet)
	}
	r.logger.Debug().Msgf("total count %d, from %d, num %d", result.TotalCount, from, num)
	if result.TotalCount > from+num {
		result.PageInfo.HasNextPage = true
		nFrom := min(from+num-1, result.TotalCount-1)
		r.logger.Debug().Msgf("next: from %d, num %d", nFrom, num)
		if result.PageInfo.EndCursor, err = NewCursor(nFrom, num).Encode(); err != nil {
			return nil, errors.Wrap(err, "cannot marshal end cursor")
		}
	}
	if from > 0 {
		result.PageInfo.HasPreviousPage = true
		nFrom := max(from-num-1, -1)
		r.logger.Debug().Msgf("prev: from %d, num %d", nFrom, num)
		if result.PageInfo.StartCursor, err = NewCursor(nFrom, num).Encode(); err != nil {
			return nil, errors.Wrap(err, "cannot marshal end cursor")
		}
	}
	result.PageInfo.CurrentCursor, err = NewCursor(from, num).Encode()
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal current cursor")
	}
	for _, hit := range resp.Hits.Hits {
		source := &sourcetype.SourceData{}
		if err := json.Unmarshal(hit.Source_, source); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal hit %v", hit)
		}
		entry := sourceToMediathekFullEntry(source)
		result.Edges = append(result.Edges, entry)
	}
	return result, nil
}

// MediathekEntries is the resolver for the mediathekEntries field.
func (r *ElasticResolver) MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, errors.Errorf("%s", errValue)
	}
	docs, err := r.loadEntries(ctx, signatures)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load entries %v", signatures)
	}

	entries := make([]*model.MediathekFullEntry, 0)
	var access = make(map[string]bool)
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, errors.Wrap(err, "cannot get groups from context")
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

func (r *ElasticResolver) ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	if errValue := ctx.Value("error"); errValue != nil {
		return nil, errors.Errorf("%s", errValue)
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
		return nil, errors.Wrapf(err, "cannot load entries %v", signatures)
	}
	groups, err := stringsFromContext(ctx, "groups")
	if err != nil {
		return nil, errors.Wrap(err, "cannot get groups from context")
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

var _ Resolver = (*ElasticResolver)(nil)
