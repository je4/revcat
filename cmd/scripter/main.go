package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var configfile = flag.String("config", "", "location of toml configuration file")
var local = flag.Bool("local", false, "run with local badger database")

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

var mediaserverUriRegexp = regexp.MustCompile(`^mediaserver:([^/]+)/(.+)$`)

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
			log.Fatalf("cannot open logfile %s: %v", conf.LogFile, err)
		}
		defer fp.Close()
		out = fp
	}

	//	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(out).With().Timestamp().Logger()
	_logger.Level(zLogger.LogLevel(conf.LogLevel))
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
		logger.Fatal().Err(err)
	}

	signature := "zotero2-2486551*"
	searchRequest := &search.Request{
		Query: &types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{
					types.Query{
						Term: map[string]types.TermQuery{
							"acl.content.keyword": types.TermQuery{
								Value: "global/guest",
							},
						},
					},
					types.Query{
						Wildcard: map[string]types.WildcardQuery{
							"signature.keyword": types.WildcardQuery{
								Value: &signature,
							},
						},
					},
				},
			},
		},
	}

	var searchAfter types.FieldValue = ""
	var num int64 = 0
	for {
		search := elastic.Search().
			Index(conf.ElasticSearch.Index).
			Request(searchRequest).
			Sort("signature.keyword").
			Size(500)
		if searchAfter != "" {
			search.SearchAfter(searchAfter)
		}
		resp, err := search.Do(context.Background())
		if err != nil {
			logger.Fatal().Err(err)
		}
		for _, hit := range resp.Hits.Hits {
			searchAfter = hit.Sort[0]
			num++

			source := &sourcetype.SourceData{}
			if err := json.Unmarshal(hit.Source_, source); err != nil {
				logger.Fatal().Err(err).Msgf("cannot unmarshal hit %v", hit)
			}
			for _, medias := range source.Media {
				for _, media := range medias {
					if strings.HasPrefix(media.Uri, "mediaserver:act/") {

					}
					if matches := mediaserverUriRegexp.FindStringSubmatch(media.Uri); matches != nil {
						fmt.Printf("%s;%s\n", matches[1], matches[2])
					}
				}
			}
		}
		if len(resp.Hits.Hits) < 500 {
			break
		}
	}

}
