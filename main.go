package main

import (
	"context"
	"fmt"
	"log"
	"mdw/health"
	"mdw/loadbalancer"
	"mdw/routes"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

type ConfiguracaoServidor struct {
	Port           string
	Handler        http.Handler
	HealthCheckURL string
	ChaosModeConfig
}

type ChaosModeConfig struct {
	Enabled      bool
	FailureRate  int
	ShutdownRate int
}

func loadEnv() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("falha ao carregar o arquivo .env: %w", err)
	}
	return nil
}

func getChaosModeConfig() ChaosModeConfig {
	config := ChaosModeConfig{
		Enabled:      os.Getenv("CHAOS_MODE_ENABLED") == "true",
		FailureRate:  35,
		ShutdownRate: 35,
	}

	if config.Enabled {
		if rate := os.Getenv("CHAOS_FAILURE_RATE"); rate != "" {
			if parsed, err := strconv.Atoi(rate); err == nil {
				config.FailureRate = parsed
			}
		}
		if rate := os.Getenv("CHAOS_SHUTDOWN_RATE"); rate != "" {
			if parsed, err := strconv.Atoi(rate); err == nil {
				config.ShutdownRate = parsed
			}
		}
	}

	return config
}

func criarServidor(config ConfiguracaoServidor) *http.Server {
	return &http.Server{
		Addr:         ":" + config.Port,
		Handler:      config.Handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func iniciarServidor(ctx context.Context, g *errgroup.Group, server *http.Server) {
	g.Go(func() error {
		log.Printf("Iniciando o servidor em %s\n", server.Addr)

		errChan := make(chan error, 1)
		go func() {
			errChan <- server.ListenAndServe()
		}()

		select {
		case <-ctx.Done():
			log.Printf("Contexto cancelado para servidor %s, iniciando shutdown...\n", server.Addr)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		case err := <-errChan:
			if err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("erro no servidor %s: %w", server.Addr, err)
			}
			return nil
		}
	})
}

func monitorarServidor(ctx context.Context, config ConfiguracaoServidor, server *http.Server) {
	restartServer := func() error {
		log.Printf("Reiniciando servidor em: %s\n", server.Addr)
		if err := server.Close(); err != nil {
			return fmt.Errorf("erro ao fechar o servidor: %w", err)
		}

		time.Sleep(1 * time.Second)

		newServer := criarServidor(config)
		iniciarServidor(ctx, &g, newServer)
		server = newServer
		return nil
	}

	var checker health.HealthChecker = health.NewHTTPChecker(health.CheckConfig{
		URL:           config.HealthCheckURL,
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	})

	if config.ChaosModeConfig.Enabled {
		checker = health.NewChaosChecker(checker, config.ChaosModeConfig.FailureRate, config.ChaosModeConfig.ShutdownRate)
	}

	monitor := health.NewMonitor(health.MonitorConfig{
		Name:          fmt.Sprintf("Server-%s", config.Port),
		CheckInterval: 30 * time.Second,
		OnUnhealthy: func() error {
			log.Println("Servidor considerado não saudável. Iniciando reinicialização ou simulação de shutdown.")

			chaosChecker := checker.(*health.ChaosChecker)
			if chaosChecker.GetServerShutdown() {
				log.Println("Simulando desligamento do servidor.")
				chaosChecker.ResetShutdownStatus()
				log.Println("Servidor será encerrado agora (simulação de shutdown).")
				os.Exit(1)
			}

			return restartServer()
		},
		MaxRetries: 3,
		RetryDelay: 5 * time.Second,
	}, checker)

	go monitor.Start(ctx)
}

var g errgroup.Group

func main() {
	if err := loadEnv(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chaosConfig := getChaosModeConfig()

	handler01, err := routes.MdwRouter_01()
	if err != nil {
		log.Fatalf("Falha ao criar o roteador 1: %v", err)
	}

	handler02, err := routes.MdwRouter_02()
	if err != nil {
		log.Fatalf("Falha ao criar o roteador 2: %v", err)
	}

	servidores := []ConfiguracaoServidor{
		{
			Port:            os.Getenv("SERVER_01_PORT"),
			Handler:         handler01,
			HealthCheckURL:  fmt.Sprintf("http://localhost:%s/health_check", os.Getenv("SERVER_01_PORT")),
			ChaosModeConfig: chaosConfig,
		},
		{
			Port:            os.Getenv("SERVER_02_PORT"),
			Handler:         handler02,
			HealthCheckURL:  fmt.Sprintf("http://localhost:%s/health_check", os.Getenv("SERVER_02_PORT")),
			ChaosModeConfig: chaosConfig,
		},
	}

	var g errgroup.Group

	for _, config := range servidores {
		server := criarServidor(config)
		cfg := config

		g.Go(func() error {
			log.Printf("Iniciando servidor na porta %s", cfg.Port)
			errChan := make(chan error, 1)

			go func() {
				errChan <- server.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				log.Printf("Desligando servidor na porta %s...", cfg.Port)
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			case err := <-errChan:
				if err != nil && err != http.ErrServerClosed {
					cancel() // Cancel context to trigger shutdown of other servers
					return fmt.Errorf("servidor na porta %s falhou: %w", cfg.Port, err)
				}
				return nil
			}
		})

		g.Go(func() error {
			monitorarServidor(ctx, cfg, server)
			return nil
		})
	}

	servers := []string{
		"http://localhost:" + os.Getenv("SERVER_01_PORT"),
		"http://localhost:" + os.Getenv("SERVER_02_PORT"),
	}
	lb := loadbalancer.NewLoadBalancer(servers)

	g.Go(func() error {
		log.Println("Iniciando o load balancer em http://localhost:8000...")
		lbServer := &http.Server{
			Addr:    ":8000",
			Handler: lb,
		}

		errChan := make(chan error, 1)
		go func() {
			errChan <- lbServer.ListenAndServe()
		}()

		select {
		case <-ctx.Done():
			log.Println("Desligando load balancer...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return lbServer.Shutdown(shutdownCtx)
		case err := <-errChan:
			if err != nil && err != http.ErrServerClosed {
				cancel()
				return fmt.Errorf("load balancer falhou: %w", err)
			}
			return nil
		}
	})

	if err := g.Wait(); err != nil {
		log.Printf("Error: %v", err)
		cancel()
		os.Exit(1)
	}
}
