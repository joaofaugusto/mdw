package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Essa função permite que crie vários servers com diferentes portas (sempre checar o .env)
func criarRouter(message string) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Recovery())
	e.GET("/", func(c *gin.Context) {
		c.JSON(
			http.StatusOK,
			gin.H{
				"codigo":   http.StatusOK,
				"mensagem": message,
			},
		)
	})

	e.GET("/health_check", HealthCheckHandler)
	return e
}

// Inicialização dos servers retornando a mensagem deles
func MdwRouter_01() http.Handler {
	return criarRouter("Servidor 01")
}

func MdwRouter_02() http.Handler {
	return criarRouter("Servidor 02")
}
