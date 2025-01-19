package FlowServer

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

func ginLoggerMiddleware(c *gin.Context) {
	ctx := c.Request.Context()

	// Start the timer
	startTime := time.Now()

	// Store the request details
	path := c.Request.URL.Path
	// raw := c.Request.URL.RawQuery  // ToDo: Get only user_id from query

	c.Next()

	// Calculate the latency
	latency := time.Since(startTime)

	// Get the status code, client IP, request method and error message from the gin context
	statusCode := c.Writer.Status()
	clientIP := c.ClientIP() // ToDo: Check if client IP logging is GDPR compliant
	userAgent := c.GetHeader("User-Agent")
	method := c.Request.Method
	errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

	// Fetch the request details
	arguments := map[string]string{
		"status_code":  strconv.Itoa(statusCode),
		"latency_time": latency.String(),
		"user_agent":   userAgent,
		"method":       method,
		"path":         path,

		// To enable investigation of possible security incidents, partially log the client IP address
		// Remove last octet of the client IP address for privacy reasons and to comply with GDPR
		"client_ip_shortened": anonymizeIP(clientIP), // ToDo: Make endpoint to activate full IP logging in runtime
	}

	jsonBytes, err := json.Marshal(arguments)
	if err != nil {
		err = errors.Wrap(err, "Failed to marshal arguments")
		logger.Error(ctx, err)
	}
	argumentsString := string(jsonBytes)

	// Log the request details
	if len(errorMessage) > 0 {
		// Insert the error message at the beginning of the arguments slice
		err := errors.New(errorMessage)

		logger.Error(ctx, err, "; Endpoint call: ", argumentsString) // ToDo: Check if eg wrong email format err is causing this too
	} else {
		logger.Debug(ctx, "Endpoint call: ", argumentsString)
	}
}
