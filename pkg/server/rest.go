package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/utils/v2/pkg/zLogger"
)

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

func (ctrl *Controller) getSignature(c *gin.Context) {
	signature := c.Param("signature")
	ctrl.logger.Info().Msgf("getSignature: signature=%s", signature)
	entries, err := ctrl.resolver.LoadEntries(context.Background(), []string{signature})
	if err != nil {
		ctrl.logger.Error().Err(err).Msgf("getSignature: signature=%s", signature)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(entries) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "signature not found"})
		return
	}
	c.JSON(http.StatusOK, entries[0])
}
