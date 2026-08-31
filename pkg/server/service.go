package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/config"
	_ "github.com/je4/revcat/v2/docs"
	"github.com/je4/revcat/v2/pkg/resolver"
	"github.com/je4/utils/v2/pkg/zLogger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

func NewController(localAddr, externalAddr string, cert *tls.Certificate, serverResolver resolver.Resolver, clients []*config.Client, syncJWTKey string, logger zLogger.ZLogger) *Controller {
	// for faster access
	clientByApiKey := make(map[string]*config.Client)
	for _, client := range clients {
		logger.Debug().Msgf("adding client %s with apikey '%s'", client.Name, client.Apikey.String())
		clientByApiKey[string(client.Apikey)] = client
	}

	ctrl := &Controller{
		localAddr:    localAddr,
		externalAddr: externalAddr,
		srv:          nil,
		cert:         cert,
		logger:       logger,
		resolver:     serverResolver,
	}
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("revcat")))

	restRouter := router.Group("/rest")
	restRouter.Use(cors.Default())
	restRouter.Use(restAuthMiddleware(syncJWTKey, logger))
	restRouter.GET("/item/:signature", ctrl.getSignature)
	restRouter.POST("/item/:signature", ctrl.updateSignature)
	restRouter.DELETE("/item/:signature", ctrl.deleteSignature)

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
	resolver     resolver.Resolver
}

func (ctrl *Controller) Handler() http.Handler {
	if ctrl.srv != nil {
		return ctrl.srv.Handler
	}
	return nil
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
