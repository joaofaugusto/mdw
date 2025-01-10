package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"mdw/routes"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}
}

func getChaosModeConfig() (bool, int, int) {
	chaosModeEnabled := os.Getenv("CHAOS_MODE_ENABLED") == "true"
	chaosFailureRate := 35  // default to 10% if not set
	chaosShutdownRate := 35 // default to 10% if not set

	if chaosModeEnabled {
		if rate := os.Getenv("CHAOS_FAILURE_RATE"); rate != "" {
			fmt.Sscanf(rate, "%d", &chaosFailureRate)
		}
		if rate := os.Getenv("CHAOS_SHUTDOWN_RATE"); rate != "" {
			fmt.Sscanf(rate, "%d", &chaosShutdownRate)
		}
	}

	return chaosModeEnabled, chaosFailureRate, chaosShutdownRate
}

func criarServidor(port, message string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func iniciarServidor(g *errgroup.Group, server *http.Server) {
	g.Go(func() error {
		log.Printf("Iniciando servidor em %s\n", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
}

func healthCheck(url string, chaosModeEnabled bool, chaosFailureRate int) bool {
	// Simulate a random failure (controlled by .env variable)
	if chaosModeEnabled && rand.Intn(100) < chaosFailureRate {
		log.Printf("Falha simulada em: %s\n", url)
		return false
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Saúde do servidor falhou em %s: %v", url, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true
	}

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Checagem de saúde do servidor falhou em %s, status code: %d, body: %s", url, resp.StatusCode, string(body))
	return false
}

func monitorAndRestartServer(server *http.Server, url string, chaosModeEnabled bool, chaosShutdownRate int, chaosFailureRate int) {
	for {
		// Simulate a random shutdown (controlled by .env variable)
		if chaosModeEnabled && rand.Intn(100) < chaosShutdownRate {
			log.Printf("Simulando queda do servidor em %s\n", server.Addr)
			server.Close()
			// Re-create and restart the server after shutting it down
			newServer := criarServidor(server.Addr[len(":"):], "Servidor Reiniciado", server.Handler)
			iniciarServidor(&g, newServer)
		}

		if !healthCheck(url, chaosModeEnabled, chaosFailureRate) {
			log.Printf("Saúde do servidor falhou em %s. Reinciando...\n", url)

			// Stop the server and restart it
			server.Close()
			// Re-create and restart the server
			newServer := criarServidor(server.Addr[len(":"):], "Servidor Reiniciado", server.Handler)
			iniciarServidor(&g, newServer)
		}

		// Wait some time before the next check
		time.Sleep(30 * time.Second)
	}
}

var (
	g errgroup.Group
)

func main() {
	// carregando o .env
	loadEnv()

	// carregando as portas do .env
	porta_server_01 := os.Getenv("SERVER_01_PORT")
	porta_server_02 := os.Getenv("SERVER_02_PORT")

	chaosModeEnabled, chaosFailureRate, chaosShutdownRate := getChaosModeConfig()

	server1 := criarServidor(porta_server_01, "Mensagem Não Usada", routes.MdwRouter_01())
	server2 := criarServidor(porta_server_02, "Mensagem Não Usada", routes.MdwRouter_02())

	iniciarServidor(&g, server1)
	iniciarServidor(&g, server2)

	// Checa as rotas e reinicia caso necessário
	go monitorAndRestartServer(server1, "http://localhost:"+porta_server_01+"/health_check", chaosModeEnabled, chaosShutdownRate, chaosFailureRate)

	go monitorAndRestartServer(server2, "http://localhost:"+porta_server_02+"/health_check", chaosModeEnabled, chaosShutdownRate, chaosFailureRate)

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
