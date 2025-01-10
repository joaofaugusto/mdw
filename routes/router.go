package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Essa função permite que crie vários servers com diferentes portas (sempre checar o .env)
func criarRouter(message string) http.Handler {
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
	return e
}

// Inicialização dos servers retornando a mensagem deles
func MdwRouter_01() http.Handler {
	return criarRouter("Servidor 01")
}

func MdwRouter_02() http.Handler {
	return criarRouter("Servidor 02")
}
