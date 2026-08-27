package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
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
