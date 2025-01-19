package FlowServer

import (
	"context"
	"github.com/LucaSchmitz2003/FlowWatch"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	tracer = otel.Tracer("ServerTracer")
	logger = FlowWatch.GetLogHelper()
)

type DefineRoutesFunc func(ctx context.Context, router *gin.Engine)

// InitServer initializes the server and returns the server address and router.
func InitServer(ctx context.Context, defineRoutes DefineRoutesFunc, acceptedOrigins []string) (string, *gin.Engine) {
	// Create a span
	ctx, span := tracer.Start(ctx, "Initialize server")
	defer span.End()

	// Set the Gin mode to release or debug
	if releaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create a new router instance with default middleware
	router := gin.New()

	// Add the server address to the accepted origins to allow same-origin requests
	acceptedOrigins = append(acceptedOrigins, serverAddress)

	// Use CORS middleware and allow given origins
	router.Use(CORSMiddleware(ctx, acceptedOrigins))

	// Load the standard gin recovery middleware
	router.Use(gin.Recovery())

	// Use the custom logger middleware
	router.Use(ginLoggerMiddleware)

	// Define the http routes for the server
	defineRoutes(ctx, router)

	// Initialize the Swagger documentation
	initSwaggerDocs(ctx, router)

	return serverAddress, router
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
