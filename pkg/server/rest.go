package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/pkg/sourcetype"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, "invalid bearer token")
			return
		}
		c.Next()
	}
}

// @Summary      Get item by signature
// @Description  Load catalog entry by signature from resolver
// @Tags         item
// @Produce      json
// @Param        signature path string true "Item Signature"
// @Security     BearerAuth
// @Success      200 {object} sourcetype.SourceData
// @Failure      401 {string} string "no authorization header / no bearer token"
// @Failure      404 {object} map[string]string "signature not found"
// @Failure      500 {object} map[string]string "internal error"
// @Router       /item/{signature} [get]
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

// @Summary      Update item by signature
// @Description  Store catalog entry for signature into resolver
// @Tags         item
// @Accept       json
// @Produce      plain
// @Param        signature path string true "Item Signature"
// @Param        data body sourcetype.SourceData true "Item Data"
// @Security     BearerAuth
// @Success      200 {string} string "object stored"
// @Failure      400 {object} map[string]string "bad request"
// @Failure      401 {string} string "unauthorized"
// @Failure      500 {object} map[string]string "internal error"
// @Router       /item/{signature} [post]
func (ctrl *Controller) updateSignature(c *gin.Context) {
	signature := c.Param("signature")
	ctrl.logger.Info().Msgf("updateSignature: signature=%s", signature)
	data := &sourcetype.SourceData{}
	if err := c.BindJSON(data); err != nil {
		ctrl.logger.Error().Err(err).Msgf("updateSignature: signature=%s", signature)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.resolver.StoreEntry(context.Background(), signature, data); err != nil {
		ctrl.logger.Error().Err(err).Msgf("updateSignature: signature=%s", signature)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.String(http.StatusOK, "object %s stored", data.Signature)
}

// @Summary      Delete item by signature
// @Description  Delete catalog entry by signature from resolver
// @Tags         item
// @Produce      plain
// @Param        signature path string true "Item Signature"
// @Security     BearerAuth
// @Success      200 {string} string "object deleted"
// @Failure      401 {string} string "unauthorized"
// @Failure      500 {object} map[string]string "internal error"
// @Router       /item/{signature} [delete]
func (ctrl *Controller) deleteSignature(c *gin.Context) {
	signature := c.Param("signature")
	ctrl.logger.Info().Msgf("deleteSignature: signature=%s", signature)
	if err := ctrl.resolver.DeleteEntry(context.Background(), signature); err != nil {
		ctrl.logger.Error().Err(err).Msgf("deleteSignature: signature=%s", signature)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.String(http.StatusOK, "object %s deleted", signature)
}
