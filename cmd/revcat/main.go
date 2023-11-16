package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/data/certs"
	"github.com/je4/revcat/v2/pkg/server"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var configfile = flag.String("config", "", "location of toml configuration file")

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

	conf := &RevCatConfig{
		LogFile:      "",
		LogLevel:     "DEBUG",
		LocalAddr:    "localhost:81",
		ExternalAddr: "http://localhost:81/graphql",
	}

	if err := LoadRevCatConfig(cfgFS, cfgFile, conf); err != nil {
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
	}
	elastic, err := elasticsearch.NewTypedClient(elasticConfig)
	if err != nil {
		logger.Fatal().Err(err)
	}

	var cert *tls.Certificate
	if conf.TLSCert != "" {
		c, err := tls.LoadX509KeyPair(conf.TLSCert, conf.TLSKey)
		if err != nil {
			logger.Fatal().Msgf("cannot load tls certificate: %v", err)
		}
		cert = &c
	} else {
		certBytes, err := fs.ReadFile(certs.CertFS, "localhost.cert.pem")
		if err != nil {
			logger.Fatal().Msgf("cannot read internal cert")
		}
		keyBytes, err := fs.ReadFile(certs.CertFS, "localhost.key.pem")
		if err != nil {
			logger.Fatal().Msgf("cannot read internal key")
		}
		c, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			logger.Fatal().Msgf("cannot create internal cert")
		}
		cert = &c
	}

	ctrl := server.NewController(conf.LocalAddr, conf.ExternalAddr, cert, elastic, conf.ElasticSearch.Index, logger)
	ctrl.Start()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	fmt.Println("press ctrl+c to stop server")
	s := <-done
	fmt.Println("got signal:", s)

	if err := ctrl.Stop(); err != nil {
		logger.Fatal().Msgf("cannot stop server: %v", err)
	}
}
