package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type LoadTesterConfig struct {
	targetURL      string
	duration       time.Duration
	requestsPerSec float64
	concurrent     int
	timeout        time.Duration
}

type LoadTesterResult struct {
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	totalLatency       time.Duration
	minLatency         time.Duration
	maxLatency         time.Duration
	statusCodes        map[int]int
	mu                 sync.Mutex
}

func main() {
	// Command line flags for configuration
	url := flag.String("url", "http://localhost:8000", "URL de destino para testar")
	duration := flag.Duration("duracao", 1*time.Minute, "Duração do teste")
	rps := flag.Float64("rps", 10, "Solicitações por segundo")
	concurrent := flag.Int("concorrencia", 5, "Numero de concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "Request timeout")
	flag.Parse()

	config := LoadTesterConfig{
		targetURL:      *url,
		duration:       *duration,
		requestsPerSec: *rps,
		concurrent:     *concurrent,
		timeout:        *timeout,
	}

	result := runLoadTest(config)
	printResults(result, config)
}

func runLoadTest(config LoadTesterConfig) *LoadTesterResult {
	result := &LoadTesterResult{
		statusCodes: make(map[int]int),
		minLatency:  time.Hour, // Start with a high value to ensure first request sets it
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.duration)
	defer cancel()

	// Create a rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.requestsPerSec), 1)

	// Create a WaitGroup to wait for all workers
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, config, limiter, result)
		}()
	}

	// Wait for completion or context cancellation
	wg.Wait()

	return result
}

func worker(ctx context.Context, config LoadTesterConfig, limiter *rate.Limiter, result *LoadTesterResult) {
	client := &http.Client{
		Timeout: config.timeout,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := limiter.Wait(ctx)
			if err != nil {
				return
			}

			startTime := time.Now()
			resp, err := client.Get(config.targetURL)
			latency := time.Since(startTime)

			result.mu.Lock()
			result.totalRequests++

			if err != nil {
				result.failedRequests++
			} else {
				result.successfulRequests++
				result.statusCodes[resp.StatusCode]++
				resp.Body.Close()

				// Update latency statistics
				result.totalLatency += latency
				if latency < result.minLatency {
					result.minLatency = latency
				}
				if latency > result.maxLatency {
					result.maxLatency = latency
				}
			}
			result.mu.Unlock()
		}
	}
}

func printResults(result *LoadTesterResult, config LoadTesterConfig) {
	fmt.Println("\nResultados do teste de carga")
	fmt.Println("================")
	fmt.Printf("URL de destino: %s\n", config.targetURL)
	fmt.Printf("Duração: %v\n", config.duration)
	fmt.Printf("RPS tentado: %.2f\n", config.requestsPerSec)
	fmt.Printf("Concurrent Workers: %d\n\n", config.concurrent)

	fmt.Printf("Total de solicitações: %d\n", result.totalRequests)
	fmt.Printf("Solicitações bem sucedidas: %d\n", result.successfulRequests)
	fmt.Printf("Solicitações com falha: %d\n", result.failedRequests)

	if result.successfulRequests > 0 {
		avgLatency := result.totalLatency / time.Duration(result.successfulRequests)
		fmt.Printf("\nEstatísticas de latência:\n")
		fmt.Printf("  Média: %v\n", avgLatency)
		fmt.Printf("  Min: %v\n", result.minLatency)
		fmt.Printf("  Max: %v\n", result.maxLatency)
	}

	fmt.Printf("\nDistribuição de código de status:\n")
	for code, count := range result.statusCodes {
		fmt.Printf("  %d: %d requisicoes\n", code, count)
	}

	actualRPS := float64(result.totalRequests) / config.duration.Seconds()
	fmt.Printf("\nRPS atingido: %.2f\n", actualRPS)
}
