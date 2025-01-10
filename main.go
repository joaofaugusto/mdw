package main

import (
	"log"
	"mdw/routes"
	"net/http"
	"os"
	"time"

	"io"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}
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

func healthCheck(url string) bool {
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

func monitorAndRestartServer(server *http.Server, url string) {
	for {
		if !healthCheck(url) {
			log.Printf("Saúde do servidor falhou em %s. Reinciando...", url)

			// Para o servidor e reinicia
			server.Close()
			// Re-cria e reinicia o servidor
			newServer := criarServidor(server.Addr[len(":"):], "Servidor Reiniciado", server.Handler)
			iniciarServidor(&g, newServer)
		}
		// Tempo entre uma checagem e outra
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

	server1 := criarServidor(porta_server_01, "Mensagem Não Usada", routes.MdwRouter_01())
	server2 := criarServidor(porta_server_02, "Mensagem Não Usada", routes.MdwRouter_02())

	iniciarServidor(&g, server1)
	iniciarServidor(&g, server2)

	// Checa as rotas e reinicia caso necessário
	go monitorAndRestartServer(server1, "http://localhost:"+porta_server_01+"/health_check")
	go monitorAndRestartServer(server2, "http://localhost:"+porta_server_02+"/health_check")

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
