package routes

import (
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

var Router = gin.Default()

func IniciarRotas() {

	Logging()

	Router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "teste",
		})
	})
	Router.Run()
}

func Logging() {
	// Desabilitando as cores do log no terminal
	gin.DisableConsoleColor()

	f, err := os.OpenFile("log/history.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("Falha ao abrir o arquivo de log: " + err.Error())
	}

	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
}
