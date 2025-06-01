package tests

import (
	"strings"
	"testing"
	"time"
)

// Estruturas de teste para mineração
type TestBlock struct {
	Index            int
	Nonce            int
	Hash             string
	PrevHash         string
	Timestamp        string
	ContainsSyra     bool
	MinerID          string
	MiningDifficulty int
	Transactions     []*TestTransaction
}

// Mock do sistema de mineração
type mockMiningSystem struct {
	difficultyTarget int
}

func newMockMiningSystem() *mockMiningSystem {
	return &mockMiningSystem{
		difficultyTarget: 3,
	}
}

func (m *mockMiningSystem) mineBlock(prevHash string, minerID string, transactions []*TestTransaction) *TestBlock {
	hash := "000" + strings.Repeat("a", 40) // Simula hash que atende dificuldade
	return &TestBlock{
		Index:            1,
		Nonce:            12345,
		Hash:             hash,
		PrevHash:         prevHash,
		Timestamp:        time.Now().Format(time.RFC3339),
		ContainsSyra:     true,
		MinerID:          minerID,
		MiningDifficulty: m.difficultyTarget,
		Transactions:     transactions,
	}
}

func (m *mockMiningSystem) validateBlock(block *TestBlock) bool {
	// Verifica se o hash começa com zeros (dificuldade)
	targetPrefix := strings.Repeat("0", m.difficultyTarget)
	return strings.HasPrefix(block.Hash, targetPrefix) && block.ContainsSyra
}

func (m *mockMiningSystem) adjustDifficulty(averageMiningTime time.Duration) {
	targetTime := 2 * time.Minute

	if averageMiningTime < targetTime/2 {
		// Blocos sendo minerados muito rapidamente
		m.difficultyTarget++
	} else if averageMiningTime > targetTime*2 {
		// Blocos sendo minerados muito lentamente
		if m.difficultyTarget > 1 {
			m.difficultyTarget--
		}
	}
}

// Testes para mineração
func TestBlockMining(t *testing.T) {
	miner := newMockMiningSystem()
	prevHash := "prevhash12345"
	minerID := "TestMiner"

	// Cria transações para o bloco
	transactions := []*TestTransaction{
		mockCreateTransaction("Alice", "Bob", 50, "transfer"),
		mockCreateTransaction("SYSTEM", minerID, 5, "mining_reward"),
	}

	block := miner.mineBlock(prevHash, minerID, transactions)

	// Verifica se o bloco foi minerado corretamente
	if !miner.validateBlock(block) {
		t.Error("Bloco minerado não passou na validação")
	}

	if block.PrevHash != prevHash {
		t.Error("PrevHash incorreto no bloco minerado")
	}

	if block.MinerID != minerID {
		t.Error("MinerID incorreto no bloco minerado")
	}

	if len(block.Transactions) != 2 {
		t.Error("Número incorreto de transações no bloco")
	}
}

func TestDifficultyAdjustment(t *testing.T) {
	miner := newMockMiningSystem()
	initialDifficulty := miner.difficultyTarget

	// Testa ajuste para blocos muito rápidos
	miner.adjustDifficulty(30 * time.Second) // Muito mais rápido que 2 minutos
	if miner.difficultyTarget <= initialDifficulty {
		t.Error("A dificuldade deveria aumentar para blocos minerados muito rapidamente")
	}

	// Testa ajuste para blocos muito lentos
	currentDifficulty := miner.difficultyTarget
	miner.adjustDifficulty(5 * time.Minute) // Muito mais lento que 2 minutos
	if miner.difficultyTarget >= currentDifficulty {
		t.Error("A dificuldade deveria diminuir para blocos minerados muito lentamente")
	}
}

func TestProofOfWorkValidation(t *testing.T) {
	miner := newMockMiningSystem()

	// Bloco válido
	validBlock := &TestBlock{
		Hash:         "000abcdef", // Tem zeros suficientes
		ContainsSyra: true,        // Contém "Syra"
	}

	if !miner.validateBlock(validBlock) {
		t.Error("Bloco válido falhou na validação")
	}

	// Bloco com hash inválido (dificuldade)
	invalidHash := &TestBlock{
		Hash:         "1abcdef", // Não tem zeros suficientes
		ContainsSyra: true,
	}

	if miner.validateBlock(invalidHash) {
		t.Error("Bloco com hash inválido não falhou na validação")
	}

	// Bloco sem "Syra"
	invalidSyra := &TestBlock{
		Hash:         "000abcdef", // Tem zeros suficientes
		ContainsSyra: false,       // Não contém "Syra"
	}

	// No nosso mock, estamos verificando apenas os zeros no início e ContainsSyra
	if miner.validateBlock(invalidSyra) {
		t.Error("Bloco sem 'Syra' não falhou na validação")
	}
}
