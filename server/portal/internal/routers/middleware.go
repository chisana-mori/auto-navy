package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Predefined username and password for Basic Auth
// TODO: Move these credentials to configuration or a more secure storage

// BasicAuthMiddleware provides Basic HTTP Authentication for accessing protected routes.
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, hasAuth := c.Request.BasicAuth()

		if hasAuth && user == BasicAuthUser && pass == BasicAuthPassword {
			// Authentication successful
			c.Next()
		} else {
			// Authentication failed
			c.Writer.Header().Set("WWW-Authenticate", BasicAuthRealm)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": MsgUnauthorized})
		}
	}
}
