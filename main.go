package main

import (
	"log"
	"mdw/routes"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

var (
	g errgroup.Group
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func createServer(port, message string, handler http.Handler) *http.Server {
	if port == "" {
		log.Fatalf("Port must be defined for %s", message)
	}
	return &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func startServer(g *errgroup.Group, server *http.Server) {
	g.Go(func() error {
		log.Printf("Starting server on %s\n", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
}

func main() {
	// carregando o .env
	loadEnv()

	// carregando as portas do .env
	porta_server_01 := ":" + os.Getenv("SERVER_01_PORT")
	porta_server_02 := ":" + os.Getenv("SERVER_02_PORT")

	server1 := createServer(porta_server_01, "Servidor 01", routes.MdwRouter_01())
	server2 := createServer(porta_server_02, "Servidor 02", routes.MdwRouter_02())

	var g errgroup.Group
	startServer(&g, server1)
	startServer(&g, server2)

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
