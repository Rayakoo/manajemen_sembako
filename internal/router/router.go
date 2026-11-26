package router

import (
	"net/http"

	"ayam-batu-api/docs"
	"ayam-batu-api/internal/handler"
	"ayam-batu-api/internal/middleware"
	"ayam-batu-api/internal/response"

	"github.com/gin-gonic/gin"
)

// Dependencies groups all HTTP handlers and middleware config for router wiring.
type Dependencies struct {
	JWTSecret      string
	AuthHandler    *handler.AuthHandler
	ProductHandler *handler.ProductHandler
	BatchHandler   *handler.BatchHandler
	SaleHandler    *handler.SaleHandler
	ReportHandler  *handler.ReportHandler
}

// Setup configures Gin routes and middleware.
func Setup(ginMode string, deps Dependencies) *gin.Engine {
	gin.SetMode(ginMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, "ok", gin.H{"status": "healthy"})
	})

	r.GET("/docs/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPISpec)
	})

	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/signup", deps.AuthHandler.SignUp)
			auth.POST("/login", deps.AuthHandler.Login)
			auth.GET("/google/login", deps.AuthHandler.GoogleLogin)
			auth.GET("/google/callback", deps.AuthHandler.GoogleCallback)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthRequired(deps.JWTSecret))
		{
			protected.GET("/auth/me", deps.AuthHandler.Me)

			products := protected.Group("/products")
			{
				products.POST("", deps.ProductHandler.Create)
				products.GET("", deps.ProductHandler.List)
				products.GET("/:id", deps.ProductHandler.GetByID)
				products.PUT("/:id", deps.ProductHandler.Update)
				products.DELETE("/:id", deps.ProductHandler.Delete)
			}

			batches := protected.Group("/batches")
			{
				batches.POST("", deps.BatchHandler.Create)
				batches.GET("", deps.BatchHandler.List)
				batches.GET("/:id", deps.BatchHandler.GetByID)
				batches.PUT("/:id", deps.BatchHandler.Update)
				batches.DELETE("/:id", deps.BatchHandler.Delete)
			}

			sales := protected.Group("/sales")
			{
				sales.POST("", deps.SaleHandler.Create)
				sales.GET("", deps.SaleHandler.List)
				sales.GET("/daily", deps.SaleHandler.DailySummary)
				sales.GET("/:id", deps.SaleHandler.GetByID)
				sales.PUT("/:id", deps.SaleHandler.Update)
				sales.DELETE("/:id", deps.SaleHandler.Delete)
			}

			reports := protected.Group("/reports")
			{
				reports.GET("/financial", deps.ReportHandler.Financial)
				reports.GET("/daily", deps.ReportHandler.Daily)
			}
		}
	}

	return r
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Ayam Batu API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: '/docs/openapi.yaml',
        dom_id: '#swagger-ui',
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>`
