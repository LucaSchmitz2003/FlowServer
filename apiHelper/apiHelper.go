package apiHelper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LucaSchmitz2003/FlowWatch/loggingHelper"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	files "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/otel"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var (
	tracer = otel.Tracer("ServerTracer")
	logger = loggingHelper.GetLogHelper()
)

type DefineRoutesFunc func(ctx context.Context, router *gin.Engine)

// InitServer initializes the server and returns the server address and router.
func InitServer(ctx context.Context, defineRoutes DefineRoutesFunc) (string, *gin.Engine) {
	// Create a span
	ctx, span := tracer.Start(ctx, "Initialize server")
	defer span.End()

	// Load the environment variables to make sure that the settings have already been loaded
	_ = godotenv.Load(".env")

	// Set the Gin mode to release
	releaseMode, err := strconv.ParseBool(os.Getenv("RELEASE_MODE"))
	if err != nil {
		err = errors.Wrap(err, "Failed to parse RELEASE_MODE, using default")
		logger.Warn(ctx, err)
		releaseMode = false
	}
	if releaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create a new router instance with default middleware
	router := gin.New()

	// Load the standard gin recovery middleware
	router.Use(gin.Recovery())

	// Use the custom logger middleware
	router.Use(ginLoggerMiddleware)

	// Define the http routes for the server
	defineRoutes(ctx, router)

	// Set up the server address
	serverName := os.Getenv("SERVER_IP")
	if serverName == "" {
		err := errors.New("SERVER_IP not set, using default")
		logger.Warn(ctx, err)
		serverName = "0.0.0.0"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		err := errors.New("SERVER_PORT not set, using default")
		logger.Warn(ctx, err)
		port = "8080"
	}

	// Initialize the Swagger documentation
	initSwaggerDocs(ctx, router)

	return fmt.Sprintf("%s:%s", serverName, port), router
}

// StartServer starts the server asynchronously and returns a keep-alive function for deferred use.
func StartServer(ctx context.Context, router *gin.Engine, address string) func() {
	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	// Channel to listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	// Context to manage server shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(ctx)

	// Run the server in a goroutine
	go func() {
		logger.Info(ctx, "Starting server on ", address)
		err := server.ListenAndServe() // ToDo: Add TLS support (ListenAndServeTLS())
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal(ctx, errors.Wrap(err, "Server encountered an error"))
		}
	}()

	// Goroutine to handle shutdown signals
	go func() {
		select {
		case <-signalChan:
			logger.Info(ctx, "Received termination signal, shutting down server...")
			shutdownCancel()
		case <-shutdownCtx.Done():
		}
	}()

	// Return a keep-alive function that waits for shutdown
	return func() {
		<-shutdownCtx.Done()

		// Graceful shutdown
		logger.Info(ctx, "Shutting down server gracefully...")

		gracefulCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(gracefulCtx)
		if err != nil {
			logger.Fatal(ctx, errors.Wrap(err, "Failed to shutdown server gracefully"))
		}

		logger.Info(ctx, "Server shutdown complete.")
	}
}

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

func anonymizeIP(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP.To4() != nil {
		// IPv4
		lastIndex := strings.LastIndex(ip, ".")
		if lastIndex != -1 {
			return ip[:lastIndex] + ".xxx"
		}
	} else if parsedIP.To16() != nil {
		// IPv6
		lastIndex := strings.LastIndex(ip, ":")
		if lastIndex != -1 {
			return ip[:lastIndex] + ":xxxx"
		}
	}
	return ip
}

// initSwaggerDocs initializes the Swagger documentation for the API using the swag tool from the correct wd.
func initSwaggerDocs(ctx context.Context, router *gin.Engine) {
	// Create a span
	ctx, span := tracer.Start(ctx, "Initialize the Swagger documentation")
	defer span.End()

	// Dynamically get the current working directory
	dir, err1 := os.Getwd()
	if err1 != nil {
		err1 = errors.Wrap(err1, "Failed to get current working directory")
		logger.Fatal(ctx, err1)
	}

	// Run the swag init command to generate the Swagger documentation
	cmd := exec.Command("swag", "init")
	cmd.Dir = dir
	err2 := cmd.Run()
	if err2 != nil {
		err2 = errors.Wrap(err2, "Failed to generate Swagger documentation")
		logger.Fatal(ctx, err2)
	}

	// Configure the Swagger-UI to explicitly use swagger.json
	url := ginSwagger.URL("/docs/swagger.json") // Tell Swagger-UI where to find the JSON file

	// Register the Swagger UI routes and redirect the /docs route to the index.html
	router.GET("/docs/*any", func(c *gin.Context) {
		if c.Request.URL.Path == "/docs" || c.Request.URL.Path == "/docs/" {
			// Redirection to index.html
			c.Redirect(302, "/docs/index.html")
			return
		} else if c.Request.URL.Path == "/docs/swagger.yaml" {
			// Directly deliver Swagger YAML
			c.File("./docs/swagger.yaml")
			return
		} else if c.Request.URL.Path == "/docs/swagger.json" {
			// Directly deliver Swagger JSON
			c.File("./docs/swagger.json")
			return
		}

		// Standard Swagger-UI handler
		ginSwagger.WrapHandler(files.Handler, url)(c)
	})
	logger.Info(ctx, "Swagger UI route registered at /docs")
}
