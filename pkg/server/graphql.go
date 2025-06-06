package server

import (
	"context"
	"crypto/tls"
	"emperror.dev/errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/revcat/v2/tools/graph"
	"github.com/je4/utils/v2/pkg/zLogger"
	"net/http"
	"strings"
	"time"
)

func graphqlHandler(serverResolver resolver.Resolver, logger zLogger.ZLogger) gin.HandlerFunc {
	h := handler.NewDefaultServer(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: graph.NewResolver(serverResolver, logger),
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

func NewController(localAddr, externalAddr string, cert *tls.Certificate, serverResolver resolver.Resolver, clients []*config.Client, logger zLogger.ZLogger) *Controller {
	// for faster access
	clientByApiKey := make(map[string]*config.Client)
	for _, client := range clients {
		clientByApiKey[string(client.Apikey)] = client
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

	checkAuthMiddleware := func() gin.HandlerFunc {
		type groupClaims struct {
			Groups string `json:"groups"`
			jwt.RegisteredClaims
		}
		return func(c *gin.Context) {
			authString := c.Request.Header.Get("Authorization")
			if authString == "" {
				logger.Info().Msg("no authorization header")
				ctx := context.WithValue(c.Request.Context(), "error", "no authorization header")
				c.Request = c.Request.WithContext(ctx)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, "no authorization header")
				return
			}
			if !strings.HasPrefix(authString, "Bearer ") {
				logger.Info().Msgf("authorization '%s' header has wrong type", authString)
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("authorization '%s' header has wrong type", authString))
				c.Request = c.Request.WithContext(ctx)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, "no bearer token")
				return
			}
			tokenString := authString[7:]
			parts := strings.SplitN(tokenString, ".", 2)

			client, ok := clientByApiKey[parts[0]]
			if !ok {
				logger.Info().Msgf("invalid application key '%s'", parts[0])
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("invalid application key '%s'", parts[0]))
				c.Request = c.Request.WithContext(ctx)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid application key")
				return

			}
			logger.Debug().Msgf("client: %s", client.Name)
			if len(parts) != 2 {
				// we only have an application key
				ctx := context.WithValue(c.Request.Context(), "groups", client.Groups)
				ctx = context.WithValue(ctx, "client", client.Name)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}

			token, err := jwt.ParseWithClaims(parts[1], &groupClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(client.JWTKey), nil
			}, jwt.WithLeeway(5*time.Second), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
			if err != nil {
				logger.Info().Err(err).Msgf("cannot parse token '%s'", tokenString)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, fmt.Sprintf("cannot parse token '%s': %v", tokenString, err))
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("cannot parse token '%s': %v", tokenString, err))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			if !token.Valid {
				logger.Info().Msgf("invalid token '%s'", tokenString)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid token")
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("invalid token '%s'", tokenString))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			claims, ok := token.Claims.(*groupClaims)
			if !ok {
				logger.Info().Msgf("invalid claims '%s'", tokenString)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid claims")
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("invalid claims '%s'", tokenString))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			exp, err := claims.GetExpirationTime()
			if err != nil {
				logger.Info().Err(err).Msgf("cannot get expiration time '%s'", tokenString)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, fmt.Sprintf("cannot get expiration time '%s': %v", tokenString, err))
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("cannot get expiration time '%s': %v", tokenString, err))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			iat, err := claims.GetIssuedAt()
			if err != nil {
				logger.Info().Err(err).Msgf("cannot get issued at time '%s'", tokenString)
				//c.AbortWithStatusJSON(http.StatusUnauthorized, fmt.Sprintf("cannot get issued at time '%s': %v", tokenString, err))
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("cannot get issued at time '%s': %v", tokenString, err))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			if iat.Time.Add(time.Duration(client.JWTMaxAge)).Before(exp.Time) {
				logger.Info().Msgf("token '%s' has more lifetime than allowed (%s)", tokenString, client.JWTMaxAge.String())
				ctx := context.WithValue(c.Request.Context(), "error", fmt.Sprintf("token '%s' has more lifetime than allowed (%s)", tokenString, client.JWTMaxAge.String()))
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return

			}
			groups := []string{}
			if strings.TrimSpace(claims.Groups) != "" {
				groups = strings.Split(claims.Groups, ";")
			}
			ctx := context.WithValue(c.Request.Context(), "groups", groups)
			ctx = context.WithValue(ctx, "client", client.Name)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}
	}

	subRouter.Use(checkAuthMiddleware())

	subRouter.POST("/", graphqlHandler(serverResolver, logger))
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
