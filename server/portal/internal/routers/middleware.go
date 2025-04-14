package routers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Predefined username and password for Basic Auth
// TODO: Move these credentials to configuration or a more secure storage
const (
	basicAuthUser = "admin"
	basicAuthPass = "password"
)

// BasicAuthMiddleware provides Basic HTTP Authentication for accessing protected routes.
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, hasAuth := c.Request.BasicAuth()

		if hasAuth && user == basicAuthUser && pass == basicAuthPass {
			// Authentication successful
			c.Next()
		} else {
			// Authentication failed
			c.Writer.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		}
	}
}