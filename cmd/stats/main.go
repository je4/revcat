package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"image"
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

const HEIGHT = 150

var configfile = flag.String("config", "", "location of toml configuration file")
var clientParam = flag.String("client", "performance", "client name")
var csvFile = flag.String("csv", "", "location of csv file")

type imgData struct {
	signature string
	img       image.Image
}

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

type statEntry struct {
	Images      int64
	Audio       int64
	Video       int64
	PDF         int64
	VideoLength int64
	AudioLength int64
	Documents   int64
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
	var stats = map[string]*statEntry{
		"total": {},
	}
	var mediaserverRegexp *regexp.Regexp = regexp.MustCompile("^mediaserver:([^/]+)/([^/]+)$")
	for {
		result, err := elastic.Search().Query(query).Sort(sort).SearchAfter(searchAfter...).Index(conf.ElasticSearch.Index).Do(context.Background())

		if err != nil {
			logger.Panic().Err(err).Msgf(err.Error())
		}
		if len(result.Hits.Hits) == 0 {
			break
		}
		for _, doc := range result.Hits.Hits {
			searchAfter = doc.Sort
			counter++

			// do all the stuff here
			jsonBytes := doc.Source_
			source := sourcetype.SourceData{ID: *doc.Id_}
			if err := json.Unmarshal(jsonBytes, &source); err != nil {
				logger.Panic().Err(err).Msg("cannot unmarshal source data")
			}
			if _, ok := stats[source.CollectionTitle]; !ok {
				stats[source.CollectionTitle] = &statEntry{}
			}
			stats[source.CollectionTitle].Documents++
			if !source.HasMedia {
				continue
			}
			for mType, mediaList := range source.Media {
				for _, media := range mediaList {
					matches := mediaserverRegexp.FindStringSubmatch(media.Uri)
					if matches == nil {
						logger.Error().Msgf("invalid url format: %s", media.Uri)
						break
					}
					//					collection := matches[1]
					//					signature := matches[2]
					logger.Info().Msgf("Loading %s", media.Uri)
					switch mType {
					case "image":
						stats[source.CollectionTitle].Images++
						stats["total"].Images++
					case "video":
						stats[source.CollectionTitle].Video++
						stats["total"].Video++
						stats[source.CollectionTitle].VideoLength += media.Duration
						stats["total"].VideoLength += media.Duration
					case "audio":
						stats[source.CollectionTitle].Audio++
						stats["total"].Audio++
						stats[source.CollectionTitle].AudioLength += media.Duration
						stats["total"].AudioLength += media.Duration
					case "pdf":
						stats[source.CollectionTitle].PDF++
						stats["total"].PDF++
					default:
						logger.Warn().Msgf("invalid media type - %s", mType)
						break
					}
				}
			}
		}
	}
	logger.Info().Msgf("found %d documents", counter)
	logger.Info().Msgf("found %d collections", len(stats))
	if *csvFile != "" {
		csvFP, err := os.Create(*csvFile)
		if err != nil {
			logger.Panic().Err(err).Msgf("cannot create csv file %s", *csvFP)
		}
		defer csvFP.Close()
		writer := csv.NewWriter(csvFP)
		defer writer.Flush()
		writer.Write([]string{"collection", "documents", "images", "audio", "video", "pdf", "video length (min)", "audio length (min)"})
		for k, v := range stats {
			if k == "total" {
				continue
			}
			writer.Write([]string{k, fmt.Sprintf("%d", v.Documents), fmt.Sprintf("%d", v.Images), fmt.Sprintf("%d", v.Audio), fmt.Sprintf("%d", v.Video), fmt.Sprintf("%d", v.PDF), fmt.Sprintf("%d", v.VideoLength/(60)), fmt.Sprintf("%d", v.AudioLength/(60))})
		}
	}
	fmt.Printf("found %d documents\n", counter)
	fmt.Printf("found %d collections\n", len(stats))

	for k, v := range stats {
		fmt.Printf("collection: %s - documents: %d - images: %d - audio: %d - video: %d - pdf: %d - video length: %dh - audio length: %dh\n", k, v.Documents, v.Images, v.Audio, v.Video, v.PDF, v.VideoLength/(60*60), v.AudioLength/(60*60))
	}
}
