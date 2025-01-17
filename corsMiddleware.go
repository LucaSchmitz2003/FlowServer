package FlowServer

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func CORSMiddleware(ctx context.Context, acceptedOrigins []string) gin.HandlerFunc {
	// Create a span
	ctx, span := tracer.Start(ctx, "Return CORS middleware")
	defer span.End()

	// Check if accepted origins contains the wildcard
	if contains(acceptedOrigins, "*") { // Abort if true, since using '*' is insecure and not supported by the middleware
		logger.Fatal(ctx, "CORS middleware accept all origins is not allowed")
		return nil
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		origin := c.Request.Header.Get("Origin")

		// Check if the given origin is accepted
		var acceptedOrigin bool
		for _, o := range acceptedOrigins {
			if strings.EqualFold(o, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				acceptedOrigin = true
				logger.Debug(ctx, "CORS allowed origin: ", origin)
				break
			}
		}

		// If origin is not accepted, abort
		if !acceptedOrigin {
			c.AbortWithStatus(http.StatusForbidden)
			logger.Warn(ctx, "Given origin is not allowed: ", origin)
			return
		} else {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Answer directly if it is an OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next() // Go to the actual request
	}
}

func contains(stringSlice []string, value string) bool {
	for _, element := range stringSlice {
		if element == value {
			return true
		}
	}
	return false
}
