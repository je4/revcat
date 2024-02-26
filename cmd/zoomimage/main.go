package main

import (
	"context"
	"crypto/tls"
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
	"golang.org/x/image/draw"
	"image"
	"image/png"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const HEIGHT = 150

var configfile = flag.String("config", "", "location of toml configuration file")
var clientParam = flag.String("client", "test", "client name")

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
	var mediaserverRegexp *regexp.Regexp = regexp.MustCompile("^mediaserver:([^/]+)/([^/]+)$")
	var images = []imgData{}
	var width int64
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
			source := sourcetype.SourceData{ID: doc.Id_}
			if err := json.Unmarshal(jsonBytes, &source); err != nil {
				logger.Panic().Err(err).Msg("cannot unmarshal source data")
			}
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
					collection := matches[1]
					signature := matches[2]
					logger.Info().Msgf("Loading %s", media.Uri)
					switch mType {
					case "image":
						if media.Mimetype == "image/x-canon-cr2" {
							logger.Warn().Msg("ignoring mime type image/x-canon-cr2")
							continue
						}
					case "video":
						signature += "$$timeshot$$3"
					case "audio":
						signature += "$$poster"
					case "pdf":
						signature += "$$poster"
					default:
						logger.Warn().Msgf("invalid media type - %s", mType)
						break
					}
					msUrl := fmt.Sprintf("%s/%s/%s/resize/autorotate/formatpng/size%d0x%d", conf.Mediaserver, collection, signature, HEIGHT, HEIGHT)
					logger.Info().Msgf("loading media: %s", msUrl)
					client := http.Client{
						Timeout: 3600 * time.Second,
					}
					resp, err := client.Get(msUrl)
					if err != nil {
						logger.Panic().Err(err).Msgf("cannot load url %s", msUrl)
					}
					defer resp.Body.Close()
					if resp.StatusCode >= 300 {
						logger.Error().Msgf("cannot get image: %v - %s", resp.StatusCode, resp.Status)
						//return errors.New(fmt.Sprintf("cannot get image: %v - %s", resp.StatusCode, resp.Status))
						break
					}
					img, _, err := image.Decode(resp.Body)
					if err != nil {
						logger.Error().Msgf("cannot decode image %s", msUrl)
						break
					}
					dst := image.NewRGBA(image.Rect(0, 0, (conf.ZoomImageHeight*img.Bounds().Max.X)/img.Bounds().Max.Y, conf.ZoomImageHeight))
					draw.ApproxBiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)
					images = append(images, imgData{signature: fmt.Sprintf("%s", source.Signature), img: dst})
					width += int64(img.Bounds().Dx())
				}

			}
		}
	}
	logger.Info().Msgf("loaded %v images", len(images))
	targetArea := float64(HEIGHT) * float64(width) * (1 + 0.05/conf.AspectRatio)
	w := math.Sqrt(conf.AspectRatio * targetArea)
	h := targetArea / w
	logger.Info().Msgf("target area: %f, width: %d, height: %d", targetArea, int(w), int(h))
	intDx := int(w)
	intDy := int(h)
	rand.Shuffle(len(images), func(i, j int) { images[i], images[j] = images[j], images[i] })
	collage := image.NewRGBA(image.Rectangle{
		Min: image.Point{},
		Max: image.Point{X: intDx, Y: intDy},
	})

	row := 0
	posX := 0
	positions := map[string][]image.Rectangle{}
	for i := 0; i < len(images); i++ {
		posY := row * conf.ZoomImageHeight
		key := i
		img := images[key]
		//	for key, img := range images {
		logger.Info().Msgf("collage image #%v of %v", key+1, len(images))
		draw.Copy(collage,
			image.Point{X: posX, Y: posY},
			img.img,
			img.img.Bounds(),
			draw.Over,
			nil)
		if _, ok := positions[img.signature]; !ok {
			positions[img.signature] = []image.Rectangle{}
		}
		positions[img.signature] = append(positions[img.signature], image.Rectangle{
			Min: image.Point{X: posX, Y: posY},
			Max: image.Point{X: posX + img.img.Bounds().Dx(), Y: posY + img.img.Bounds().Dy()},
		})
		posX += img.img.Bounds().Max.X
		if posX > intDx {
			posX = 0
			row++
			// repeat cropped image
			i--
		}
		if (row+1)*conf.ZoomImageHeight > intDy {
			logger.Info().Msgf("collage %v images of %v", key+1, len(images))
			break
		}
	}
	fp, err := os.Create(filepath.Join(conf.CollagePath, "collage.png"))
	if err != nil {
		logger.Panic().Err(err).Msg("cannot create collage file")
	}
	logger.Info().Msgf("encoding collage: %d x %d", intDx, intDy)
	if err := png.Encode(fp, collage); err != nil {
		fp.Close()
		logger.Panic().Err(err).Msg("cannot encode collage png")
	}
	fp.Close()

	fp, err = os.Create(filepath.Join(conf.CollagePath, "collage.json"))
	if err != nil {
		logger.Panic().Err(err).Msg("cannot create collage json file")
	}
	jsonW := json.NewEncoder(fp)
	if err := jsonW.Encode(positions); err != nil {
		fp.Close()
		logger.Panic().Err(err).Msg("cannot marshal json")
	}
	fp.Close()
	fp, err = os.Create(filepath.Join(conf.CollagePath, "collage.jsonl"))
	if err != nil {
		logger.Panic().Err(err).Msg("cannot create collage jsonl file")
	}
	for signature, rects := range positions {
		jsonBytes, err := json.Marshal(map[string]interface{}{
			"signature": signature,
			"rects":     rects,
		})
		if err != nil {
			fp.Close()
			logger.Panic().Err(err).Msg("cannot marshal JSONL")
		}
		jsonBytes = append(jsonBytes, []byte("\n")...)
		if _, err := fp.Write(jsonBytes); err != nil {
			fp.Close()
			logger.Panic().Msg("cannot store JSONL")
		}
	}
	fp.Close()
	fp, err = os.Create(filepath.Join(conf.CollagePath, "signatures.txt"))
	if err != nil {
		logger.Panic().Err(err).Msg("cannot create signatures file")
	}
	for signature, _ := range positions {
		str := signature + "\n"
		if _, err := fp.Write([]byte(str)); err != nil {
			fp.Close()
			logger.Panic().Err(err).Msg("cannot store signatures")
		}
	}
	fp.Close()

}
