package FlowServer

import (
	"context"
	"github.com/gin-gonic/gin"
	files "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// initSwaggerDocs provides the Swagger documentation for the API.
func initSwaggerDocs(ctx context.Context, router *gin.Engine) {
	// Create a span
	ctx, span := tracer.Start(ctx, "Initialize the Swagger documentation")
	defer span.End()

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
