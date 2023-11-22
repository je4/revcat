package server

import (
	"context"
	"crypto/tls"
	"emperror.dev/errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/tools/graph"
	"github.com/je4/utils/v2/pkg/zLogger"
	"net/http"
	"strings"
	"time"
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

func NewController(localAddr, externalAddr string, cert *tls.Certificate, elastic *elasticsearch.TypedClient, index string, clients []config.Client, logger zLogger.ZLogger) *Controller {
	// for faster access
	clientByApiKey := make(map[string]config.Client)
	for _, client := range clients {
		clientByApiKey[client.Apikey] = client
	}

	ctrl := &Controller{
		localAddr:    localAddr,
		externalAddr: externalAddr,
		srv:          nil,
		cert:         cert,
		logger:       logger,
	}
	router := gin.Default()

	subRouter := router.Group("/graphql")

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	subRouter.Use(cors.New(corsConfig))

	/*
		// add the gin context to the request context
		GinContextToContextMiddleware := func() gin.HandlerFunc {
			return func(c *gin.Context) {
				ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			}
		}
	*/
	checkAuthMiddleware := func() gin.HandlerFunc {
		type groupClaims struct {
			Groups []string `json:"groups"`
			jwt.RegisteredClaims
		}
		return func(c *gin.Context) {
			authString := c.Request.Header.Get("Authorization")
			if authString == "" {
				logger.Info().Msg("no authorization header")
				c.AbortWithStatusJSON(http.StatusUnauthorized, "no authorization header")
				return
			}
			if !strings.HasPrefix(authString, "Bearer ") {
				logger.Info().Msgf("authorization '%s' header has wrong type", authString)
				c.AbortWithStatusJSON(http.StatusUnauthorized, "no bearer token")
				return
			}
			tokenString := authString[7:]
			parts := strings.SplitN(tokenString, ".", 2)
			client, ok := clientByApiKey[parts[0]]
			if !ok {
				logger.Info().Msgf("invalid application key '%s'", parts[0])
				c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid application key")
				return

			}
			if len(parts) != 2 {
				// we only have an application key
				ctx := context.WithValue(c.Request.Context(), "groups", client.Groups)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}

			token, err := jwt.ParseWithClaims(tokenString, &groupClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(client.JWTSecret), nil
			}, jwt.WithLeeway(5*time.Second))
			if err != nil {
				logger.Info().Err(err).Msgf("cannot parse token '%s'", tokenString)
				c.AbortWithStatusJSON(http.StatusUnauthorized, fmt.Sprintf("cannot parse token '%s': %v", tokenString, err))
				return
			}
			if !token.Valid {
				logger.Info().Msgf("invalid token '%s'", tokenString)
				c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid token")
				return
			}
			claims, ok := token.Claims.(*groupClaims)
			if !ok {
				logger.Info().Msgf("invalid claims '%s'", tokenString)
				c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid claims")
				return
			}
			ctx := context.WithValue(c.Request.Context(), "groups", claims.Groups)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}
	subRouter.Use(checkAuthMiddleware())

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
