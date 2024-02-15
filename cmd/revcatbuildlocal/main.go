package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/dgraph-io/badger/v4"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
)

var configfile = flag.String("config", "", "location of toml configuration file")

type LoggingHttpElasticClient struct {
	c http.Client
}

func (l LoggingHttpElasticClient) RoundTrip(request *http.Request) (*http.Response, error) {
	// Log the http request dump
	requestDump, err := httputil.DumpRequest(request, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("reqDump: " + string(requestDump))
	return l.c.Do(request)
}

func main() {
	flag.Parse()

	var cfgFS fs.FS
	var cfgFile string
	if *configfile != "" {
		cfgFS = os.DirFS(filepath.Dir(*configfile))
		cfgFile = filepath.Base(*configfile)
	} else {
		cfgFS = config.ConfigFS
		cfgFile = "revcat.toml"
	}

	conf := &config.RevCatConfig{
		LogFile:      "",
		LogLevel:     "DEBUG",
		LocalAddr:    "localhost:81",
		ExternalAddr: "http://localhost:81/graphql",
		Client:       []*config.Client{},
		ElasticSearch: config.ElasticSearchConfig{
			Debug: false,
		},
	}

	if err := config.LoadRevCatConfig(cfgFS, cfgFile, conf); err != nil {
		log.Fatalf("cannot load toml from [%v] %s: %v", cfgFS, cfgFile, err)
	}

	// create logger instance
	var out io.Writer = os.Stdout
	if conf.LogFile != "" {
		fp, err := os.OpenFile(conf.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Panicf("cannot open logfile %s: %v", conf.LogFile, err)
		}
		defer fp.Close()
		out = fp
	}

	//	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(out).With().Timestamp().Logger()
	switch strings.ToUpper(conf.LogLevel) {
	case "DEBUG":
		_logger = _logger.Level(zerolog.DebugLevel)
	case "INFO":
		_logger = _logger.Level(zerolog.InfoLevel)
	case "WARN":
		_logger = _logger.Level(zerolog.WarnLevel)
	case "ERROR":
		_logger = _logger.Level(zerolog.ErrorLevel)
	case "FATAL":
		_logger = _logger.Level(zerolog.FatalLevel)
	case "PANIC":
		_logger = _logger.Level(zerolog.PanicLevel)
	default:
		_logger = _logger.Level(zerolog.DebugLevel)
	}
	var logger zLogger.ZLogger = &_logger

	elasticConfig := elasticsearch.Config{
		APIKey:    string(conf.ElasticSearch.ApiKey),
		Addresses: conf.ElasticSearch.Endpoint,

		// Retry on 429 TooManyRequests statuses
		//
		RetryOnStatus: []int{502, 503, 504, 429},

		// Retry up to 5 attempts
		//
		MaxRetries: 5,

		Logger: &elastictransport.ColorLogger{Output: os.Stdout},
		//		Transport: doer,
	}
	if conf.ElasticSearch.Debug {
		doer := LoggingHttpElasticClient{
			c: http.Client{
				// Load a trusted CA here, if running in production
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			},
		}
		elasticConfig.Transport = doer
	}
	elastic, err := elasticsearch.NewTypedClient(elasticConfig)
	if err != nil {
		logger.Panic().Err(err)
	}

	db, err := badger.Open(badger.DefaultOptions(conf.Badger))
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	var query = &types.Query{
		Bool: &types.BoolQuery{
			Boost:              nil,
			Filter:             nil,
			MinimumShouldMatch: 1,
			Must:               []types.Query{},
			MustNot:            nil,
			QueryName_:         nil,
			Should:             []types.Query{},
		},
		Boosting:          nil,
		CombinedFields:    nil,
		Common:            nil,
		ConstantScore:     nil,
		DisMax:            nil,
		DistanceFeature:   nil,
		Exists:            nil,
		FieldMaskingSpan:  nil,
		FunctionScore:     nil,
		Fuzzy:             nil,
		GeoBoundingBox:    nil,
		GeoDistance:       nil,
		GeoPolygon:        nil,
		GeoShape:          nil,
		HasChild:          nil,
		HasParent:         nil,
		Ids:               nil,
		Intervals:         nil,
		Match:             nil,
		MatchAll:          nil,
		MatchBoolPrefix:   nil,
		MatchNone:         nil,
		MatchPhrase:       nil,
		MatchPhrasePrefix: nil,
		MoreLikeThis:      nil,
		MultiMatch:        nil,
		Nested:            nil,
		ParentId:          nil,
		Percolate:         nil,
		Pinned:            nil,
		Prefix:            nil,
		QueryString:       nil,
		Range:             nil,
		RankFeature:       nil,
		Regexp:            nil,
		RuleQuery:         nil,
		Script:            nil,
		ScriptScore:       nil,
		Shape:             nil,
		SimpleQueryString: nil,
		SpanContaining:    nil,
		SpanFirst:         nil,
		SpanMulti:         nil,
		SpanNear:          nil,
		SpanNot:           nil,
		SpanOr:            nil,
		SpanTerm:          nil,
		SpanWithin:        nil,
		Term:              nil,
		Terms:             nil,
		TermsSet:          nil,
		TextExpansion:     nil,
		Type:              nil,
		Wildcard:          nil,
		Wrapper:           nil,
	}
	for _, client := range conf.Client {
		for _, vals := range client.OR {
			for _, val := range vals.Values {
				query.Bool.Should = append(query.Bool.Should, types.Query{
					Terms: &types.TermsQuery{
						TermsQuery: map[string]types.TermsQueryField{vals.Field: []types.FieldValue{val}},
					},
				})
			}
		}
	}
	query.Bool.Must = []types.Query{}
	for _, client := range conf.Client {

		query.Bool.Must = append(query.Bool.Must, types.Query{
			Terms: &types.TermsQuery{
				TermsQuery: map[string]types.TermsQueryField{"acl.meta.keyword": client.Groups},
			},
		},
			types.Query{
				Terms: &types.TermsQuery{
					TermsQuery: map[string]types.TermsQueryField{"acl.content.keyword": client.Groups},
				},
			})
	}
	var sort = types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"_score": types.FieldSort{
				Order: &sortorder.Desc},
			"signature.keyword": types.FieldSort{
				Order: &sortorder.Asc},
		},
	}
	var searchAfter = []types.FieldValue{}
	var counter int64
	for {
		result, err := elastic.Search().Query(query).Sort(sort).SearchAfter(searchAfter...).Index(conf.ElasticSearch.Index).Do(context.Background())

		if err != nil {
			logger.Panic().Err(err).Msgf(err.Error())
		}
		if len(result.Hits.Hits) == 0 {
			break
		}
		db.Update(func(txn *badger.Txn) error {
			for _, doc := range result.Hits.Hits {
				/*
					source := sourcetype.SourceData{ID: doc.Id_}
					if err := json.Unmarshal(doc.Source_, &source); err != nil {
						logger.Panic().Err(err)
					}

				*/
				buf := bytes.NewBuffer([]byte{})
				bw := brotli.NewWriter(buf)
				if _, err := bw.Write(doc.Source_); err != nil {
					logger.Panic().Err(err)
				}
				if err := bw.Close(); err != nil {
					logger.Panic().Err(err)

				}
				if err := txn.Set([]byte(doc.Id_), buf.Bytes()); err != nil {
					logger.Panic().Err(err)
				}
				logger.Info().Msgf("[%05d]: %s", counter+1, doc.Id_)
				counter++
				searchAfter = doc.Sort
			}
			return nil
		})
	}
}
