package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/resolver"
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
var clientParam = flag.String("client", "test", "client name")

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

	conf := &config.ZoomConfig{
		LogLevel: "DEBUG",
		ElasticSearch: config.ElasticSearchConfig{
			Debug: false,
		},
	}

	if err := config.LoadZoomConfig(cfgFS, cfgFile, conf); err != nil {
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

	var client *config.Client
	for _, c := range conf.Client {
		if c.Name == *clientParam {
			client = c
			break
		}
	}
	if client == nil {
		logger.Panic().Msgf("client %s not found in config file", *clientParam)
	}

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

	baseQueries, err := resolver.BuildBaseFilter(client)
	if err != nil {
		logger.Panic().Err(err).Msg("cannot build base filter")
	}

	var query = &types.Query{
		Bool: &types.BoolQuery{
			Filter: baseQueries,
		},
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
		for _, doc := range result.Hits.Hits {
			// do all the stuff here

			searchAfter = doc.Sort
			counter++
		}

	}
}
