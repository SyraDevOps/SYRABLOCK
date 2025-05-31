package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"
)

type Validator struct {
    UserID      string `json:"user_id"`
    Stake       int    `json:"stake"`
    Address     string `json:"address"`
    Reputation  int    `json:"reputation"`
    IsActive    bool   `json:"is_active"`
    LastValidation time.Time `json:"last_validation"`
}

type ConsensusRound struct {
    RoundID        string      `json:"round_id"`
    BlockHash      string      `json:"block_hash"`
    SelectedValidator string   `json:"selected_validator"`
    Validators     []Validator `json:"validators"`
    Timestamp      time.Time   `json:"timestamp"`
    Confirmed      bool        `json:"confirmed"`
}

type StakePool struct {
    Validators map[string]*Validator `json:"validators"`
    TotalStake int                   `json:"total_stake"`
    MinStake   int                   `json:"min_stake"`
}

func loadStakePool() *StakePool {
    file, err := os.Open("../stake_pool.json")
    if err != nil {
        return &StakePool{
            Validators: make(map[string]*Validator),
            TotalStake: 0,
            MinStake:   10, // Mínimo 10 SYRA para ser validador
        }
    }
    defer file.Close()

    var pool StakePool
    json.NewDecoder(file).Decode(&pool)
    return &pool
}

func (sp *StakePool) saveStakePool() error {
    file, err := os.Create("../stake_pool.json")
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(sp)
}

func (sp *StakePool) addValidator(userID string, stake int, address string) error {
    if stake < sp.MinStake {
        return fmt.Errorf("stake mínimo é %d SYRA", sp.MinStake)
    }

    validator := &Validator{
        UserID:     userID,
        Stake:      stake,
        Address:    address,
        Reputation: 100, // Reputação inicial
        IsActive:   true,
        LastValidation: time.Now(),
    }

    sp.Validators[userID] = validator
    sp.TotalStake += stake
    return nil
}

func (sp *StakePool) selectValidator(blockHash string) *Validator {
    if len(sp.Validators) == 0 {
        return nil
    }

    // Algoritmo de seleção baseado em stake + reputação
    var candidates []struct {
        validator *Validator
        weight    float64
    }

    for _, validator := range sp.Validators {
        if !validator.IsActive {
            continue
        }

        // Peso = stake * (reputação/100) * fator temporal
        timeFactor := 1.0
        if time.Since(validator.LastValidation) < time.Hour {
            timeFactor = 0.5 // Reduz chance se validou recentemente
        }

        weight := float64(validator.Stake) * (float64(validator.Reputation) / 100.0) * timeFactor
        candidates = append(candidates, struct {
            validator *Validator
            weight    float64
        }{validator, weight})
    }

    if len(candidates) == 0 {
        return nil
    }

    // Ordena por peso
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].weight > candidates[j].weight
    })

    // Seleciona randomicamente entre os top 3 candidatos
    maxCandidates := 3
    if len(candidates) < maxCandidates {
        maxCandidates = len(candidates)
    }

    // Usa hash do bloco como seed para determinismo
    hash := sha256.Sum256([]byte(blockHash))
    seed := int64(hash[0])<<56 | int64(hash[1])<<48 | int64(hash[2])<<40 | int64(hash[3])<<32
    randGen := rand.New(rand.NewSource(seed))

    selectedIndex := randGen.Intn(maxCandidates)
    return candidates[selectedIndex].validator
}

func (sp *StakePool) processValidation(validatorID string, success bool) {
    validator, exists := sp.Validators[validatorID]
    if !exists {
        return
    }

    validator.LastValidation = time.Now()

    if success {
        validator.Reputation += 1
        if validator.Reputation > 200 {
            validator.Reputation = 200 // Cap máximo
        }
    } else {
        validator.Reputation -= 5
        if validator.Reputation < 0 {
            validator.Reputation = 0
            validator.IsActive = false // Desativa validador com reputação zero
        }
    }
}

func performConsensus(blockHash string) *ConsensusRound {
    pool := loadStakePool()
    
    // Seleciona validador
    selectedValidator := pool.selectValidator(blockHash)
    if selectedValidator == nil {
        return nil
    }

    round := &ConsensusRound{
        RoundID:           fmt.Sprintf("ROUND_%d", time.Now().UnixNano()),
        BlockHash:         blockHash,
        SelectedValidator: selectedValidator.UserID,
        Timestamp:         time.Now(),
        Confirmed:         false,
    }

    // Coleta todos os validadores para o round
    for _, validator := range pool.Validators {
        if validator.IsActive {
            round.Validators = append(round.Validators, *validator)
        }
    }

    // Simula processo de consenso (em implementação real seria distribuído)
    fmt.Printf("🔄 Consenso iniciado para bloco %s\n", blockHash[:16])
    fmt.Printf("   Validador selecionado: %s (Stake: %d, Reputação: %d)\n", 
        selectedValidator.UserID, selectedValidator.Stake, selectedValidator.Reputation)

    // Simula validação
    time.Sleep(time.Millisecond * 100) // Simula tempo de processamento
    
    // Em uma implementação real, outros validadores verificariam o trabalho
    confirmations := 0
    requiredConfirmations := len(round.Validators) * 2 / 3 // 2/3 de maioria
    
    for _, validator := range round.Validators {
        if validator.UserID != selectedValidator.UserID {
            // Simula verificação por outros validadores
            if rand.Float64() > 0.1 { // 90% de chance de confirmação
                confirmations++
            }
        }
    }

    if confirmations >= requiredConfirmations {
        round.Confirmed = true
        pool.processValidation(selectedValidator.UserID, true)
        fmt.Printf("   ✅ Consenso APROVADO (%d/%d confirmações)\n", confirmations, requiredConfirmations)
    } else {
        pool.processValidation(selectedValidator.UserID, false)
        fmt.Printf("   ❌ Consenso REJEITADO (%d/%d confirmações)\n", confirmations, requiredConfirmations)
    }

    pool.saveStakePool()
    
    // Salva round de consenso
    roundFile, err := os.Create(fmt.Sprintf("../consensus_round_%d.json", time.Now().Unix()))
    if err == nil {
        defer roundFile.Close()
        encoder := json.NewEncoder(roundFile)
        encoder.SetIndent("", "  ")
        encoder.Encode(round)
    }

    return round
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Uso: go run pos_consensus.go <comando> [parametros]")
        fmt.Println("Comandos:")
        fmt.Println("  add_validator <user_id> <stake> <address> - Adiciona validador")
        fmt.Println("  consensus <block_hash>                   - Executa consenso")
        fmt.Println("  pool_status                              - Status do pool")
        return
    }

    switch os.Args[1] {
    case "add_validator":
        if len(os.Args) < 5 {
            fmt.Println("Erro: informe user_id, stake e address")
            return
        }
        userID := os.Args[2]
        var stake int
        fmt.Sscanf(os.Args[3], "%d", &stake)
        address := os.Args[4]

        pool := loadStakePool()
        err := pool.addValidator(userID, stake, address)
        if err != nil {
            fmt.Printf("Erro: %v\n", err)
            return
        }

        pool.saveStakePool()
        fmt.Printf("Validador %s adicionado com stake de %d SYRA\n", userID, stake)

    case "consensus":
        if len(os.Args) < 3 {
            fmt.Println("Erro: informe o hash do bloco")
            return
        }
        blockHash := os.Args[2]
        
        round := performConsensus(blockHash)
        if round == nil {
            fmt.Println("Nenhum validador disponível")
            return
        }

    case "pool_status":
        pool := loadStakePool()
        fmt.Println("=== STATUS DO POOL DE VALIDADORES ===")
        fmt.Printf("Total de Stake: %d SYRA\n", pool.TotalStake)
        fmt.Printf("Stake Mínimo: %d SYRA\n", pool.MinStake)
        fmt.Printf("Validadores Ativos: %d\n", len(pool.Validators))
        
        for _, validator := range pool.Validators {
            status := "INATIVO"
            if validator.IsActive {
                status = "ATIVO"
            }
            fmt.Printf("  %s | Stake: %d | Reputação: %d | Status: %s\n",
                validator.UserID, validator.Stake, validator.Reputation, status)
        }

    default:
        fmt.Println("Comando não reconhecido")
    }
}