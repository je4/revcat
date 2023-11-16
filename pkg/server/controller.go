package server

import (
	"context"
	"crypto/tls"
	"emperror.dev/errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/je4/revcat/v2/tools/graph"
	"github.com/je4/utils/v2/pkg/zLogger"
	"net/http"
)

func graphqlHandler(elastic *elasticsearch.TypedClient, index string, logger zLogger.ZLogger) gin.HandlerFunc {
	h := handler.NewDefaultServer(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: graph.NewResolver(elastic, index, logger),
			}))
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/graphql")
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func NewController(localAddr, externalAddr string, cert *tls.Certificate, elastic *elasticsearch.TypedClient, index string, logger zLogger.ZLogger) *Controller {
	ctrl := &Controller{
		localAddr:    localAddr,
		externalAddr: externalAddr,
		srv:          nil,
		cert:         cert,
		logger:       logger,
	}
	/*
		u, err := url.Parse(ctrl.externalAddr)
		if err != nil {
			return errors.Wrapf(err, "invalid external address '%ctrl'", ctrl.externalAddr)
		}
		subpath := "/" + strings.Trim(u.Path, "/")

			// programmatically set swagger info
			docs.SwaggerInfo.Host = fmt.Sprintf("%ctrl:%ctrl", u.Hostname(), u.Port())
			docs.SwaggerInfo.BasePath = "/" + strings.Trim(subpath+BASEPATH, "/")

			if ctrl.cert == nil {
				docs.SwaggerInfo.Schemes = []string{"http"}
			} else {
				docs.SwaggerInfo.Schemes = []string{"https"}
			}
	*/
	router := gin.Default()

	subRouter := router.Group("/graphql")
	subRouter.POST("/", graphqlHandler(elastic, index, logger))
	subRouter.GET("/", playgroundHandler())

	var tlsConfig *tls.Config
	if ctrl.cert != nil {
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{*ctrl.cert},
		}
	}
	ctrl.srv = &http.Server{
		Addr:      ctrl.localAddr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}
	return ctrl
}

type Controller struct {
	localAddr    string
	externalAddr string
	srv          *http.Server
	cert         *tls.Certificate
	logger       zLogger.ZLogger
}

func (ctrl *Controller) Start() error {
	go func() {
		if ctrl.srv.TLSConfig == nil {
			fmt.Printf("starting server at http://%s\n", ctrl.localAddr)
			if err := ctrl.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				// unexpected error. port in use?
				fmt.Errorf("server on '%s' ended: %v", ctrl.localAddr, err)
			}
		} else {
			fmt.Printf("starting server at https://%s\n", ctrl.localAddr)
			if err := ctrl.srv.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
				// unexpected error. port in use?
				fmt.Errorf("server on '%s' ended: %v", ctrl.localAddr, err)
			}
		}
		// always returns error. ErrServerClosed on graceful close
	}()

	return nil
}

func (ctrl *Controller) Stop() error {
	return ctrl.srv.Shutdown(context.Background())
}
