package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type DifficultyStats struct {
	CurrentDifficulty int           `json:"current_difficulty"`
	AverageBlockTime  time.Duration `json:"average_block_time"`
	TargetBlockTime   time.Duration `json:"target_block_time"`
	LastAdjustment    time.Time     `json:"last_adjustment"`
	EfficiencyPercent float64       `json:"efficiency_percent"`
	TotalAdjustments  int           `json:"total_adjustments"`
}

func MonitorDifficulty() {
	fmt.Println("🎯 Monitor de Dificuldade Dinâmica")
	fmt.Println("==================================")

	for {
		// Carrega estatísticas atuais
		stats := LoadDifficultyStats()

		// Exibe status atual
		fmt.Printf("\r🎯 Dificuldade: %d | Tempo Médio: %.1fs | Alvo: %.1fs | Eficiência: %.1f%%",
			stats.CurrentDifficulty,
			stats.AverageBlockTime.Seconds(),
			stats.TargetBlockTime.Seconds(),
			stats.EfficiencyPercent)

		// Aguarda 5 segundos antes da próxima atualização
		time.Sleep(5 * time.Second)
	}
}

func LoadDifficultyStats() DifficultyStats {
	// Carrega configuração de dificuldade
	file, err := os.Open("../difficulty_config.json")
	if err != nil {
		return DifficultyStats{}
	}
	defer file.Close()

	var config map[string]interface{}
	json.NewDecoder(file).Decode(&config)

	// Calcula estatísticas básicas
	currentDifficulty := int(config["current_difficulty"].(float64))
	targetTime := 2 * time.Minute // 2 minutos padrão

	// Carrega últimos blocos para calcular tempo médio
	avgTime := CalculateAverageBlockTime()

	efficiency := 100.0
	if avgTime > 0 {
		efficiency = float64(targetTime) / float64(avgTime) * 100.0
	}

	return DifficultyStats{
		CurrentDifficulty: currentDifficulty,
		AverageBlockTime:  avgTime,
		TargetBlockTime:   targetTime,
		EfficiencyPercent: efficiency,
	}
}

func CalculateAverageBlockTime() time.Duration {
	// Carrega tokens para calcular tempo médio
	file, err := os.Open("../tokens.json")
	if err != nil {
		return 0
	}
	defer file.Close()

	var tokens []map[string]interface{}
	json.NewDecoder(file).Decode(&tokens)

	if len(tokens) < 2 {
		return 0
	}

	// Pega últimos 10 blocos
	start := len(tokens) - 10
	if start < 0 {
		start = 0
	}

	recentTokens := tokens[start:]

	var totalTime float64
	count := 0

	for i := 1; i < len(recentTokens); i++ {
		if miningTime, ok := recentTokens[i]["mining_time"].(float64); ok {
			totalTime += miningTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgSeconds := totalTime / float64(count)
	return time.Duration(avgSeconds * float64(time.Second))
}

// PrintStats exibe estatísticas formatadas
func PrintStats() {
	stats := LoadDifficultyStats()
	fmt.Println("=== ESTATÍSTICAS DE DIFICULDADE ===")
	fmt.Printf("Dificuldade Atual: %d\n", stats.CurrentDifficulty)
	fmt.Printf("Tempo Médio de Bloco: %.1fs\n", stats.AverageBlockTime.Seconds())
	fmt.Printf("Tempo Alvo: %.1fs\n", stats.TargetBlockTime.Seconds())
	fmt.Printf("Eficiência: %.1f%%\n", stats.EfficiencyPercent)
}
