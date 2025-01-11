package health

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type ChaosChecker struct {
	wrapped                  HealthChecker
	taxaFalha                int
	ultimoRestart            time.Time
	tempoMinimoReset         time.Duration
	falhasConsecutivas       int
	maximoFalhasConsecutivas int
	quedaSimulada            bool // Acompanha se o desligamento foi simulado
	taxaQueda                int  // Probabilidade de simular desligamento
}

func NewChaosChecker(checker HealthChecker, failureRate int, shutdownRate int) *ChaosChecker {
	return &ChaosChecker{
		wrapped:                  checker,
		taxaFalha:                failureRate,     // Inicializa a probabilidade de falha
		tempoMinimoReset:         1 * time.Minute, // Tempo minimo entre reset do caos
		ultimoRestart:            time.Now(),
		maximoFalhasConsecutivas: 3,            // Numero maximo de falhas consecutivas
		taxaQueda:                shutdownRate, // Inicializa a probabilidade de queda
	}
}

func (c *ChaosChecker) Check(ctx context.Context) error {
	// Primeiro verifique se o servico esta ok
	if err := c.wrapped.Check(ctx); err != nil {
		c.falhasConsecutivas = 0 // Redefinir o caos em caso de falha real
		return fmt.Errorf("verificação de integridade falhou: %w", err)
	}

	// aplicar somente se o tempo minimo tiver passado
	if time.Since(c.ultimoRestart) < c.tempoMinimoReset {
		return nil
	}

	// aplicar com probabilidade decrescente para falhas consecutivas
	adjustedRate := c.taxaFalha / (c.falhasConsecutivas + 1)
	if rand.Intn(100) < adjustedRate {
		c.falhasConsecutivas++
		if c.falhasConsecutivas > c.maximoFalhasConsecutivas {
			c.falhasConsecutivas = 0 // Redefinir apos o maximo de falhas consecutivas
			return nil
		}
		return fmt.Errorf("chaos mode: falha simulada (tentativa %d)", c.falhasConsecutivas)
	}

	// Verifica a simulação de queda
	if rand.Intn(100) < c.taxaQueda {
		c.quedaSimulada = true
		return fmt.Errorf("chaos mode: queda simulada")
	}

	c.falhasConsecutivas = 0
	return nil
}

// ChaosChecker agora tem um metodo para acionar o desligamento do servidor
func (c *ChaosChecker) GetServerShutdown() bool {
	// Retorna true se a queda precisar ser simulada
	return rand.Intn(100) < c.taxaFalha
}

// Redefinir o status de desligamento após acionar uma queda
func (c *ChaosChecker) ResetShutdownStatus() {
	// Redefinir qualquer estado interno ou sinalizadores aqui, se necessário
}

func (c *ChaosChecker) GetStatus() Status {
	return c.wrapped.GetStatus()
}
