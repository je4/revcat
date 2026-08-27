package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/revcat/v2/tools/graph"
	"github.com/je4/utils/v2/pkg/zLogger"
)

type groupClaims struct {
	Groups string `json:"groups"`
	jwt.RegisteredClaims
}

func setContextValue(c *gin.Context, key string, val any) {
	ctx := context.WithValue(c.Request.Context(), key, val)
	c.Request = c.Request.WithContext(ctx)
}

func extractBearerToken(authString string, logger zLogger.ZLogger) (string, error) {
	if authString == "" {
		logger.Info().Msg("no authorization header")
		return "", errors.New("no authorization header")
	}
	if !strings.HasPrefix(authString, "Bearer ") {
		logger.Info().Msgf("authorization '%s' header has wrong type", authString)
		return "", fmt.Errorf("authorization '%s' header has wrong type", authString)
	}
	return authString[7:], nil
}

func validateJWT[T jwt.Claims](tokenString, key string, claims T, maxAge time.Duration, logger zLogger.ZLogger) (T, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	}, jwt.WithLeeway(5*time.Second), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		logger.Info().Err(err).Msgf("cannot parse token '%s'", tokenString)
		return claims, fmt.Errorf("cannot parse token '%s': %w", tokenString, err)
	}
	if !token.Valid {
		logger.Info().Msgf("invalid token '%s'", tokenString)
		return claims, fmt.Errorf("invalid token '%s'", tokenString)
	}
	c, ok := token.Claims.(T)
	if !ok {
		logger.Info().Msgf("invalid claims '%s'", tokenString)
		return claims, fmt.Errorf("invalid claims '%s'", tokenString)
	}
	exp, err := c.GetExpirationTime()
	if err != nil || exp == nil {
		logger.Info().Err(err).Msgf("cannot get expiration time '%s'", tokenString)
		return claims, fmt.Errorf("cannot get expiration time '%s': %v", tokenString, err)
	}
	iat, err := c.GetIssuedAt()
	if err != nil || iat == nil {
		logger.Info().Err(err).Msgf("cannot get issued at time '%s'", tokenString)
		return claims, fmt.Errorf("cannot get issued at time '%s': %v", tokenString, err)
	}
	if maxAge > 0 && iat.Time.Add(maxAge).Before(exp.Time) {
		logger.Info().Msgf("token '%s' has more lifetime than allowed (%s)", tokenString, maxAge.String())
		return claims, fmt.Errorf("token '%s' has more lifetime than allowed (%s)", tokenString, maxAge.String())
	}
	return c, nil
}

func restAuthMiddleware(syncJWTKey string, logger zLogger.ZLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authString := c.Request.Header.Get("Authorization")
		tokenString, err := extractBearerToken(authString, logger)
		if err != nil {
			setContextValue(c, "error", err.Error())
			if authString == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, "no authorization header")
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, "no bearer token")
			}
			return
		}

		if _, err := validateJWT(tokenString, syncJWTKey, &jwt.RegisteredClaims{}, 4*time.Hour, logger); err != nil {
			setContextValue(c, "error", err.Error())
			c.Next()
			return
		}
		c.Next()
	}
}

func graphqlAuthMiddleware(clientByApiKey map[string]*config.Client, logger zLogger.ZLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.Request.Header.Get("Authorization"), logger)
		if err != nil {
			setContextValue(c, "error", err.Error())
			return
		}
		parts := strings.SplitN(tokenString, ".", 2)

		client, ok := clientByApiKey[parts[0]]
		if !ok {
			logger.Info().Msgf("invalid application key '%s'", parts[0])
			setContextValue(c, "error", fmt.Sprintf("invalid application key '%s'", parts[0]))
			return
		}
		logger.Debug().Msgf("client: %s", client.Name)
		if len(parts) != 2 {
			// we only have an application key
			setContextValue(c, "groups", client.Groups)
			setContextValue(c, "client", client.Name)
			c.Next()
			return
		}

		claims, err := validateJWT(parts[1], string(client.JWTKey), &groupClaims{}, time.Duration(client.JWTMaxAge), logger)
		if err != nil {
			setContextValue(c, "error", err.Error())
			c.Next()
			return
		}
		groups := []string{}
		if strings.TrimSpace(claims.Groups) != "" {
			groups = strings.Split(claims.Groups, ";")
		}
		setContextValue(c, "groups", groups)
		setContextValue(c, "client", client.Name)
		c.Next()
	}
}

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

func NewController(localAddr, externalAddr string, cert *tls.Certificate, serverResolver resolver.Resolver, clients []*config.Client, syncJWTKey string, logger zLogger.ZLogger) *Controller {
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
	restRouter := router.Group("/rest")
	restRouter.Use(cors.Default())
	restRouter.Use(restAuthMiddleware(syncJWTKey, logger))

	subRouter := router.Group("/graphql")
	subRouter.Use(cors.Default())
	subRouter.Use(graphqlAuthMiddleware(clientByApiKey, logger))

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
