package main

import (
	"log"
	"mdw/routes"
)

func main() {
	log.Print("Teste")
	routes.IniciarRotas()

	routes.Router.Run()
}
