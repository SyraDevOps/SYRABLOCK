package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// DifficultyManager gerencia a dificuldade dinâmica da rede
type DifficultyManager struct {
	CurrentDifficulty    int           `json:"current_difficulty"`
	TargetBlockTime      time.Duration `json:"target_block_time"`     // Tempo alvo entre blocos (ex: 2 minutos)
	DifficultyAdjustment int           `json:"difficulty_adjustment"` // A cada quantos blocos ajustar
	MaxDifficultyChange  float64       `json:"max_difficulty_change"` // Máxima variação por ajuste (ex: 25%)
	MinDifficulty        int           `json:"min_difficulty"`        // Dificuldade mínima
	MaxDifficulty        int           `json:"max_difficulty"`        // Dificuldade máxima
	LastAdjustmentBlock  int           `json:"last_adjustment_block"`
	RecentBlockTimes     []time.Time   `json:"recent_block_times"` // Tempos dos últimos blocos
}

// Estrutura para histórico de dificuldade
type DifficultyHistory struct {
	BlockIndex       int           `json:"block_index"`
	Difficulty       int           `json:"difficulty"`
	ActualBlockTime  time.Duration `json:"actual_block_time"`
	TargetBlockTime  time.Duration `json:"target_block_time"`
	AdjustmentReason string        `json:"adjustment_reason"`
	Timestamp        time.Time     `json:"timestamp"`
}

// NewDifficultyManager cria um novo gerenciador de dificuldade
func NewDifficultyManager() *DifficultyManager {
	return &DifficultyManager{
		CurrentDifficulty:    4,               // Dificuldade inicial (número de zeros necessários)
		TargetBlockTime:      2 * time.Minute, // 2 minutos por bloco
		DifficultyAdjustment: 10,              // Ajusta a cada 10 blocos
		MaxDifficultyChange:  0.25,            // Máximo 25% de variação
		MinDifficulty:        1,               // Mínimo 1 zero
		MaxDifficulty:        8,               // Máximo 8 zeros
		LastAdjustmentBlock:  0,
		RecentBlockTimes:     make([]time.Time, 0),
	}
}

// LoadDifficultyManager carrega configurações salvas
func LoadDifficultyManager() *DifficultyManager {
	file, err := os.Open("../difficulty_config.json")
	if err != nil {
		// Se não existe, cria novo
		dm := NewDifficultyManager()
		dm.Save()
		return dm
	}
	defer file.Close()

	var dm DifficultyManager
	if err := json.NewDecoder(file).Decode(&dm); err != nil {
		fmt.Printf("⚠️ Erro ao carregar configuração de dificuldade: %v\n", err)
		return NewDifficultyManager()
	}

	return &dm
}

// Save salva a configuração atual
func (dm *DifficultyManager) Save() error {
	file, err := os.Create("../difficulty_config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dm)
}

// AddBlockTime registra o tempo de um novo bloco
func (dm *DifficultyManager) AddBlockTime(blockTime time.Time, blockIndex int) {
	dm.RecentBlockTimes = append(dm.RecentBlockTimes, blockTime)

	// Mantém apenas os últimos blocos necessários para cálculo
	maxHistory := dm.DifficultyAdjustment + 5
	if len(dm.RecentBlockTimes) > maxHistory {
		dm.RecentBlockTimes = dm.RecentBlockTimes[len(dm.RecentBlockTimes)-maxHistory:]
	}

	// Verifica se é hora de ajustar dificuldade
	if blockIndex-dm.LastAdjustmentBlock >= dm.DifficultyAdjustment {
		dm.AdjustDifficulty(blockIndex)
	}
}

// AdjustDifficulty ajusta a dificuldade baseada no desempenho recente
func (dm *DifficultyManager) AdjustDifficulty(currentBlockIndex int) {
	if len(dm.RecentBlockTimes) < dm.DifficultyAdjustment {
		return // Não há dados suficientes
	}

	// Calcula tempo médio entre os últimos blocos
	recentBlocks := dm.RecentBlockTimes[len(dm.RecentBlockTimes)-dm.DifficultyAdjustment:]

	var totalTime time.Duration
	for i := 1; i < len(recentBlocks); i++ {
		blockTime := recentBlocks[i].Sub(recentBlocks[i-1])
		totalTime += blockTime
	}

	averageBlockTime := totalTime / time.Duration(len(recentBlocks)-1)

	// Calcula o fator de ajuste
	targetTime := dm.TargetBlockTime
	ratio := float64(averageBlockTime) / float64(targetTime)

	oldDifficulty := dm.CurrentDifficulty
	adjustmentReason := ""

	// Determina novo nível de dificuldade
	if ratio > 1.5 { // Blocos muito lentos
		// Diminui dificuldade (mais fácil minerar)
		change := math.Min(dm.MaxDifficultyChange, (ratio-1.0)*0.5)
		newDifficulty := float64(dm.CurrentDifficulty) * (1.0 - change)
		dm.CurrentDifficulty = int(math.Max(float64(dm.MinDifficulty), newDifficulty))
		adjustmentReason = fmt.Sprintf("Blocos muito lentos (%.1fs vs %.1fs alvo)",
			averageBlockTime.Seconds(), targetTime.Seconds())

	} else if ratio < 0.5 { // Blocos muito rápidos
		// Aumenta dificuldade (mais difícil minerar)
		change := math.Min(dm.MaxDifficultyChange, (1.0-ratio)*0.5)
		newDifficulty := float64(dm.CurrentDifficulty) * (1.0 + change)
		dm.CurrentDifficulty = int(math.Min(float64(dm.MaxDifficulty), newDifficulty))
		adjustmentReason = fmt.Sprintf("Blocos muito rápidos (%.1fs vs %.1fs alvo)",
			averageBlockTime.Seconds(), targetTime.Seconds())

	} else {
		// Ajuste fino baseado na diferença
		if ratio > 1.1 { // Ligeiramente lento
			dm.CurrentDifficulty = int(math.Max(float64(dm.MinDifficulty), float64(dm.CurrentDifficulty)*0.95))
			adjustmentReason = "Ajuste fino: ligeiramente lento"
		} else if ratio < 0.9 { // Ligeiramente rápido
			dm.CurrentDifficulty = int(math.Min(float64(dm.MaxDifficulty), float64(dm.CurrentDifficulty)*1.05))
			adjustmentReason = "Ajuste fino: ligeiramente rápido"
		} else {
			adjustmentReason = "Sem ajuste necessário"
		}
	}

	// Registra o ajuste se houve mudança
	if dm.CurrentDifficulty != oldDifficulty {
		fmt.Printf("🎯 Dificuldade ajustada: %d → %d (%s)\n",
			oldDifficulty, dm.CurrentDifficulty, adjustmentReason)

		// Salva histórico
		dm.saveAdjustmentHistory(currentBlockIndex, oldDifficulty, averageBlockTime, adjustmentReason)
	}

	dm.LastAdjustmentBlock = currentBlockIndex
	dm.Save()
}

// GetCurrentDifficulty retorna a dificuldade atual
func (dm *DifficultyManager) GetCurrentDifficulty() int {
	return dm.CurrentDifficulty
}

// GetDifficultyTarget retorna o padrão necessário para o hash
func (dm *DifficultyManager) GetDifficultyTarget() string {
	target := ""
	for i := 0; i < dm.CurrentDifficulty; i++ {
		target += "0"
	}
	return target
}

// IsValidHash verifica se o hash atende à dificuldade atual
func (dm *DifficultyManager) IsValidHash(hash string) bool {
	target := dm.GetDifficultyTarget()
	return len(hash) >= len(target) && hash[:len(target)] == target
}

// saveAdjustmentHistory salva histórico de ajustes
func (dm *DifficultyManager) saveAdjustmentHistory(blockIndex, oldDifficulty int, actualTime time.Duration, reason string) {
	history := DifficultyHistory{
		BlockIndex:       blockIndex,
		Difficulty:       dm.CurrentDifficulty,
		ActualBlockTime:  actualTime,
		TargetBlockTime:  dm.TargetBlockTime,
		AdjustmentReason: reason,
		Timestamp:        time.Now(),
	}

	// Carrega histórico existente
	var historyList []DifficultyHistory
	file, err := os.Open("../difficulty_history.json")
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&historyList)
	}

	// Adiciona novo registro
	historyList = append(historyList, history)

	// Mantém apenas os últimos 100 registros
	if len(historyList) > 100 {
		historyList = historyList[len(historyList)-100:]
	}

	// Salva histórico atualizado
	file, err = os.Create("../difficulty_history.json")
	if err == nil {
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encoder.Encode(historyList)
	}
}

// GetStats retorna estatísticas da dificuldade
func (dm *DifficultyManager) GetStats() map[string]interface{} {
	var avgTime time.Duration
	if len(dm.RecentBlockTimes) > 1 {
		var total time.Duration
		for i := 1; i < len(dm.RecentBlockTimes); i++ {
			total += dm.RecentBlockTimes[i].Sub(dm.RecentBlockTimes[i-1])
		}
		avgTime = total / time.Duration(len(dm.RecentBlockTimes)-1)
	}

	return map[string]interface{}{
		"current_difficulty":      dm.CurrentDifficulty,
		"target_block_time":       dm.TargetBlockTime,
		"average_block_time":      avgTime,
		"difficulty_target":       dm.GetDifficultyTarget(),
		"blocks_until_adjustment": dm.DifficultyAdjustment - (len(dm.RecentBlockTimes) - dm.LastAdjustmentBlock),
		"recent_blocks_count":     len(dm.RecentBlockTimes),
	}
}

// Comando principal para testar e gerenciar dificuldade
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run difficulty.go <comando> [parametros]")
		fmt.Println("Comandos:")
		fmt.Println("  stats                    - Mostra estatísticas atuais")
		fmt.Println("  monitor                  - Monitor em tempo real")
		fmt.Println("  history                  - Mostra histórico de ajustes")
		fmt.Println("  test <hash>              - Testa se hash atende dificuldade")
		fmt.Println("  simulate <block_time>    - Simula tempo de bloco (segundos)")
		fmt.Println("  reset                    - Reseta configurações")
		return
	}

	dm := LoadDifficultyManager()

	switch os.Args[1] {
	case "stats":
		stats := dm.GetStats()
		fmt.Println("=== ESTATÍSTICAS DE DIFICULDADE ===")
		for key, value := range stats {
			fmt.Printf("%s: %v\n", key, value)
		}

	case "monitor":
		// Chama função do difficulty_monitor.go
		MonitorDifficulty()

	case "history":
		file, err := os.Open("../difficulty_history.json")
		if err != nil {
			fmt.Println("Nenhum histórico encontrado")
			return
		}
		defer file.Close()

		var history []DifficultyHistory
		json.NewDecoder(file).Decode(&history)

		fmt.Println("=== HISTÓRICO DE AJUSTES ===")
		for _, h := range history {
			fmt.Printf("Bloco %d: Dificuldade %d | Tempo Real: %.1fs | Alvo: %.1fs | %s\n",
				h.BlockIndex, h.Difficulty, h.ActualBlockTime.Seconds(),
				h.TargetBlockTime.Seconds(), h.AdjustmentReason)
		}

	case "test":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o hash para testar")
			return
		}
		hash := os.Args[2]
		if dm.IsValidHash(hash) {
			fmt.Printf("✅ Hash válido para dificuldade %d\n", dm.CurrentDifficulty)
		} else {
			fmt.Printf("❌ Hash inválido para dificuldade %d (precisa começar com %s)\n",
				dm.CurrentDifficulty, dm.GetDifficultyTarget())
		}

	case "simulate":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o tempo do bloco em segundos")
			return
		}
		var seconds float64
		fmt.Sscanf(os.Args[2], "%f", &seconds)

		blockTime := time.Now().Add(-time.Duration(seconds) * time.Second)
		dm.AddBlockTime(blockTime, len(dm.RecentBlockTimes)+1)

		fmt.Printf("Simulação adicionada: bloco com tempo de %.1fs\n", seconds)
		stats := dm.GetStats()
		fmt.Printf("Nova dificuldade: %v\n", stats["current_difficulty"])

	case "reset":
		newDm := NewDifficultyManager()
		newDm.Save()
		fmt.Println("Configurações de dificuldade resetadas")

	default:
		fmt.Println("Comando não reconhecido")
	}
}
