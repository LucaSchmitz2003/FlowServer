package FlowServer

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"os"
	"strconv"
)

var (
	releaseMode   bool
	serverAddress string
)

func init() {
	ctx := context.Background()

	// Load the environment variables
	err1 := godotenv.Load(".env")
	if err1 != nil {
		err1 = errors.Wrap(err1, "Failed to load .env file")
		logger.Warn(ctx, err1)
	}

	// Get the release mode
	var err2 error
	releaseMode, err2 = strconv.ParseBool(os.Getenv("RELEASE_MODE"))
	if err2 != nil {
		err2 = errors.Wrap(err2, "Failed to parse RELEASE_MODE, using default")
		logger.Warn(ctx, err2)
		releaseMode = false
	}

	// Get the server address
	serverName := os.Getenv("DOMAIN")
	if serverName == "" {
		err := errors.New("DOMAIN not set, using default")
		logger.Warn(ctx, err)
		serverName = "0.0.0.0"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		err := errors.New("SERVER_PORT not set, using default")
		logger.Warn(ctx, err)
		port = "8080"
	}
	serverAddress = fmt.Sprintf("%s:%s", serverName, port)
}
