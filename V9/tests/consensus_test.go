package tests

import (
	"testing"
	"time"
)

// Estruturas de teste para consenso PoS
type TestValidator struct {
	ID              string
	Stake           int
	Reputation      int
	IsActive        bool
	LastValidation  time.Time
	SuccessfulVotes int
	FailedVotes     int
}

type ConsensusRound struct {
	ID            string
	BlockHash     string
	Validators    []*TestValidator
	Votes         map[string]bool
	RequiredVotes int
	Status        string
	StartTime     time.Time
}

// Mock do sistema de consenso
type mockConsensusSystem struct {
	validators map[string]*TestValidator
	rounds     map[string]*ConsensusRound
}

func newMockConsensusSystem() *mockConsensusSystem {
	return &mockConsensusSystem{
		validators: make(map[string]*TestValidator),
		rounds:     make(map[string]*ConsensusRound),
	}
}

func (cs *mockConsensusSystem) registerValidator(id string, stake int) *TestValidator {
	validator := &TestValidator{
		ID:              id,
		Stake:           stake,
		Reputation:      100, // Reputação inicial
		IsActive:        true,
		LastValidation:  time.Now().Add(-24 * time.Hour), // Última validação há 24h
		SuccessfulVotes: 0,
		FailedVotes:     0,
	}
	cs.validators[id] = validator
	return validator
}

func (cs *mockConsensusSystem) selectValidators(blockHash string, count int) []*TestValidator {
	var result []*TestValidator

	// Seleciona todos os validadores ativos
	for _, v := range cs.validators {
		if v.IsActive && v.Stake >= 10 { // Mínimo 10 tokens para validar
			result = append(result, v)
		}
	}

	// Simplificado: apenas retorna os primeiros 'count' validadores
	// Em um sistema real, usaria o hash do bloco como seed para seleção determinística
	if len(result) > count {
		result = result[:count]
	}

	return result
}

func (cs *mockConsensusSystem) startConsensusRound(blockHash string) *ConsensusRound {
	validators := cs.selectValidators(blockHash, 5)

	round := &ConsensusRound{
		ID:            "round_" + time.Now().Format(time.RFC3339),
		BlockHash:     blockHash,
		Validators:    validators,
		Votes:         make(map[string]bool),
		RequiredVotes: (len(validators) * 2 / 3) + 1, // 2/3 + 1
		Status:        "PENDING",
		StartTime:     time.Now(),
	}

	cs.rounds[round.ID] = round
	return round
}

func (cs *mockConsensusSystem) vote(roundID, validatorID string, approve bool) bool {
	round, exists := cs.rounds[roundID]
	if !exists {
		return false
	}

	// Verifica se é um validador do round
	isValidator := false
	for _, v := range round.Validators {
		if v.ID == validatorID {
			isValidator = true
			break
		}
	}

	if !isValidator {
		return false
	}

	// Registra voto
	round.Votes[validatorID] = approve

	// Atualiza estatísticas do validador
	if validator, exists := cs.validators[validatorID]; exists {
		validator.LastValidation = time.Now()
		if approve {
			validator.SuccessfulVotes++
		} else {
			validator.FailedVotes++
		}
	}

	// Verifica se há votos suficientes para aprovar
	approveCount := 0
	for _, vote := range round.Votes {
		if vote {
			approveCount++
		}
	}

	if approveCount >= round.RequiredVotes {
		round.Status = "APPROVED"
		return true
	}

	// Verifica se já não é possível aprovar
	remainingVotes := len(round.Validators) - len(round.Votes)
	if approveCount+remainingVotes < round.RequiredVotes {
		round.Status = "REJECTED"
	}

	return round.Status == "APPROVED"
}

func (cs *mockConsensusSystem) updateReputation(validatorID string, success bool) {
	validator, exists := cs.validators[validatorID]
	if !exists {
		return
	}

	if success {
		validator.Reputation += 1
		if validator.Reputation > 200 {
			validator.Reputation = 200 // Cap máximo
		}
	} else {
		validator.Reputation -= 5
		if validator.Reputation < 0 {
			validator.Reputation = 0
			validator.IsActive = false // Desativa com reputação zero
		}
	}
}

// Testes para consenso
func TestValidatorRegistration(t *testing.T) {
	cs := newMockConsensusSystem()

	validator := cs.registerValidator("Alice", 50)
	if validator.ID != "Alice" || validator.Stake != 50 {
		t.Error("Registro de validador falhou")
	}

	// Verifica validador com stake insuficiente
	lowStakeVal := cs.registerValidator("LowStake", 5)
	validators := cs.selectValidators("blockhash", 10)

	found := false
	for _, v := range validators {
		if v.ID == lowStakeVal.ID {
			found = true
			break
		}
	}

	if found {
		t.Error("Validador com stake insuficiente não deveria ser selecionado")
	}
}

func TestConsensusRound(t *testing.T) {
	cs := newMockConsensusSystem()

	// Registra validadores
	cs.registerValidator("Alice", 50)
	cs.registerValidator("Bob", 30)
	cs.registerValidator("Charlie", 40)
	cs.registerValidator("Dave", 25)
	cs.registerValidator("Eve", 35)

	// Inicia round de consenso
	round := cs.startConsensusRound("blockhash123")

	if len(round.Validators) != 5 {
		t.Error("Número incorreto de validadores selecionados")
	}

	if round.RequiredVotes != 4 { // 2/3 * 5 + 1 = 4.33 -> 4
		t.Errorf("RequiredVotes incorreto: esperado %d, obtido %d", 4, round.RequiredVotes)
	}

	// Testa votação
	cs.vote(round.ID, "Alice", true)
	cs.vote(round.ID, "Bob", true)
	cs.vote(round.ID, "Charlie", true)

	// Com 3 votos positivos e necessários 4, ainda não aprovou
	if round.Status == "APPROVED" {
		t.Error("Round aprovado com votos insuficientes")
	}

	// Voto final para aprovar
	result := cs.vote(round.ID, "Dave", true)
	if !result || round.Status != "APPROVED" {
		t.Error("Round deveria ser aprovado com 4 votos")
	}
}

func TestReputationSystem(t *testing.T) {
	cs := newMockConsensusSystem()

	// Registra validador com reputação padrão (100)
	validator := cs.registerValidator("Validator1", 50)

	// Testa aumento de reputação com validação bem-sucedida
	initialRep := validator.Reputation
	cs.updateReputation("Validator1", true)
	if validator.Reputation <= initialRep {
		t.Error("Reputação deveria aumentar após validação bem-sucedida")
	}

	// Testa penalidade severa por falhas
	initialRep = validator.Reputation
	for i := 0; i < 10; i++ {
		cs.updateReputation("Validator1", false)
	}

	if validator.Reputation >= initialRep {
		t.Error("Reputação deveria diminuir após múltiplas validações falhas")
	}

	// Testa desativação com reputação zero
	for validator.Reputation > 0 {
		cs.updateReputation("Validator1", false)
	}

	if validator.IsActive {
		t.Error("Validador com reputação zero deveria ser desativado")
	}
}

func TestConsensusTimeout(t *testing.T) {
	cs := newMockConsensusSystem()

	// Registra validadores mas apenas 2 votam
	cs.registerValidator("Alice", 50)
	cs.registerValidator("Bob", 30)
	cs.registerValidator("Charlie", 40)
	cs.registerValidator("Dave", 25)
	cs.registerValidator("Eve", 35)

	// Inicia round de consenso
	round := cs.startConsensusRound("blockhash123")

	// Apenas 2 votos (insuficiente)
	cs.vote(round.ID, "Alice", true)
	cs.vote(round.ID, "Bob", true)

	// Em um cenário real, teríamos um timeout, mas no teste
	// simulamos verificando o estado final diretamente
	if round.Status == "APPROVED" {
		t.Error("Round não deveria ser aprovado com votos insuficientes")
	}
}
