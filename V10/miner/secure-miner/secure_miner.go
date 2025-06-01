package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Transaction representa uma transação na blockchain
type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	PublicKey string    `json:"public_key"`
	Nonce     int       `json:"nonce"`
	Hash      string    `json:"hash"`
	Signature string    `json:"signature"`
	Contract  string    `json:"contract,omitempty"`
}

// calculateHash calcula o hash da transação
func (tx *Transaction) calculateHash() (string, error) {
	record := fmt.Sprintf("%s%s%s%s%d%d%s%d",
		tx.ID, tx.Type, tx.From, tx.To, tx.Amount,
		tx.Timestamp.UnixNano(), tx.PublicKey, tx.Nonce)
	h := sha256.New()
	_, err := h.Write([]byte(record))
	if err != nil {
		return "", err
	}
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed), nil
}

// P2PNode representa um nó da rede P2P
type P2PNode struct {
	ID string `json:"id"`
}

// Token representa um bloco minerado
type Token struct {
	Index        int           `json:"index"`
	Nonce        int           `json:"nonce"`
	Hash         string        `json:"hash"`
	HashParts    []string      `json:"hash_parts"`
	Timestamp    string        `json:"timestamp"`
	ContainsSyra bool          `json:"contains_syra"`
	PrevHash     string        `json:"prev_hash,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	MinerID      string        `json:"miner_id,omitempty"`
}

// TransactionValidator valida transações
type TransactionValidator struct {
	// Cache para armazenar chaves públicas validadas
	keyCache map[string]bool
}

// NewTransactionValidator cria um novo validador de transações
func NewTransactionValidator() *TransactionValidator {
	return &TransactionValidator{
		keyCache: make(map[string]bool),
	}
}

// VerifySignature verifica a assinatura da transação
func (v *TransactionValidator) VerifySignature(tx *Transaction) bool {
	// Verifica campos obrigatórios
	if tx.ID == "" || tx.From == "" || tx.Signature == "" {
		fmt.Printf("❌ Transação %s: campos obrigatórios faltando\n", tx.ID)
		return false
	}

	// Para transações do sistema (recompensas de mineração)
	if tx.From == "SYSTEM" && tx.Type == "mining_reward" {
		// Validação especial para transações do sistema
		expectedSig := "SYSTEM_SIGNATURE_" + tx.Hash[:16]
		if tx.Signature == expectedSig {
			fmt.Printf("✅ Transação do sistema %s válida\n", tx.ID)
			return true
		}
		fmt.Printf("❌ Transação do sistema %s com assinatura inválida\n", tx.ID)
		return false
	}

	// Para transações regulares, implementar verificação RSA real aqui
	// Por enquanto, simulação básica
	if len(tx.Signature) > 10 && tx.PublicKey != "" {
		fmt.Printf("✅ Transação %s: assinatura válida (simulado)\n", tx.ID)
		return true
	}

	fmt.Printf("❌ Transação %s: assinatura inválida\n", tx.ID)
	return false
}

// ValidateTransactionChain valida uma cadeia de transações
func (v *TransactionValidator) ValidateTransactionChain(txs []Transaction) bool {
	usedNonces := make(map[string]map[int]bool) // userID -> nonce -> usado

	for i, tx := range txs {
		// Verifica assinatura individual
		if !v.VerifySignature(&tx) {
			fmt.Printf("❌ Transação %d falhou na validação de assinatura\n", i)
			return false
		}

		// Verifica replay attack (nonce duplicado) apenas para transações normais
		if tx.From != "SYSTEM" {
			if usedNonces[tx.From] == nil {
				usedNonces[tx.From] = make(map[int]bool)
			}

			if usedNonces[tx.From][tx.Nonce] {
				fmt.Printf("❌ Replay attack detectado: nonce %d já usado por %s\n", tx.Nonce, tx.From)
				return false
			}

			usedNonces[tx.From][tx.Nonce] = true
		}

		// Validações específicas por tipo
		if !v.validateTransactionType(&tx) {
			return false
		}
	}

	return true
}

// validateTransactionType valida regras específicas por tipo de transação
func (v *TransactionValidator) validateTransactionType(tx *Transaction) bool {
	switch tx.Type {
	case "transfer":
		if tx.Amount <= 0 {
			fmt.Printf("❌ Transação %s: valor inválido %d\n", tx.ID, tx.Amount)
			return false
		}
		if tx.From == tx.To {
			fmt.Printf("❌ Transação %s: não pode transferir para si mesmo\n", tx.ID)
			return false
		}

	case "mining_reward":
		if tx.From != "SYSTEM" {
			fmt.Printf("❌ Transação %s: recompensa deve vir do sistema\n", tx.ID)
			return false
		}
		if tx.Amount <= 0 || tx.Amount > 10 {
			fmt.Printf("❌ Transação %s: recompensa inválida %d\n", tx.ID, tx.Amount)
			return false
		}

	case "contract":
		if tx.Contract == "" {
			fmt.Printf("❌ Transação %s: ID do contrato obrigatório\n", tx.ID)
			return false
		}

	default:
		fmt.Printf("❌ Transação %s: tipo inválido %s\n", tx.ID, tx.Type)
		return false
	}

	return true
}

// createMiningReward cria transação de recompensa assinada pelo sistema
func createMiningReward(minerID string, amount int) (*Transaction, error) {
	// Transação de recompensa especial (não precisa de chave privada real do SYSTEM)
	tx := &Transaction{
		ID:        fmt.Sprintf("REWARD_%d_%s", time.Now().UnixNano(), minerID),
		Type:      "mining_reward",
		From:      "SYSTEM",
		To:        minerID,
		Amount:    amount,
		Timestamp: time.Now(),
		PublicKey: "SYSTEM_PUBLIC_KEY", // Chave especial do sistema
		Nonce:     int(time.Now().UnixNano() % 1000000),
	}

	// Calcula hash
	hash, err := tx.calculateHash()
	if err != nil {
		return nil, err
	}
	tx.Hash = hash

	// Assinatura especial do sistema (em produção seria HSM/chave segura)
	tx.Signature = "SYSTEM_SIGNATURE_" + hash[:16]

	return tx, nil
}

// mineBlockWithValidation minera bloco incluindo apenas transações válidas
func mineBlockWithValidation(node *P2PNode, pendingTxs []Transaction) *Token {
	validator := NewTransactionValidator()
	var validTxs []Transaction

	fmt.Printf("🔍 Validando %d transações pendentes...\n", len(pendingTxs))

	// Filtra apenas transações com assinaturas válidas
	for _, tx := range pendingTxs {
		if validator.VerifySignature(&tx) {
			validTxs = append(validTxs, tx)
		} else {
			fmt.Printf("⚠️ Ignorando transação com assinatura inválida: %s\n", tx.ID)
		}
	}

	// Adiciona recompensa de mineração
	reward, err := createMiningReward(node.ID, 1)
	if err == nil {
		validTxs = append(validTxs, *reward)
		fmt.Printf("💰 Recompensa de mineração adicionada: %s\n", reward.ID)
	} else {
		fmt.Printf("⚠️ Erro ao criar recompensa de mineração: %v\n", err)
	}

	fmt.Printf("⛏️ Minerando bloco com %d transações válidas\n", len(validTxs))

	// TODO: Implementar lógica real de mineração aqui
	// Por enquanto, retorna um token placeholder
	token := &Token{
		Index:        1, // TODO: calcular índice correto
		Nonce:        0, // TODO: calcular nonce através de proof-of-work
		Hash:         "placeholder_hash",
		HashParts:    []string{"part1", "part2", "part3", "part4"},
		Timestamp:    time.Now().Format(time.RFC3339),
		ContainsSyra: true,
		Transactions: validTxs,
		MinerID:      node.ID,
	}

	return token
}

// Exemplo de uso e teste
func main() {
	fmt.Println("🔧 Testando Secure Miner...")

	// Cria um nó P2P de exemplo
	node := &P2PNode{ID: "miner_test_001"}

	// Cria algumas transações de exemplo
	pendingTxs := []Transaction{
		{
			ID:        "TX_001",
			Type:      "transfer",
			From:      "Alice",
			To:        "Bob",
			Amount:    50,
			Timestamp: time.Now(),
			PublicKey: "alice_public_key",
			Nonce:     1,
			Hash:      "tx_hash_001",
			Signature: "valid_signature_001",
		},
		{
			ID:        "TX_002",
			Type:      "transfer",
			From:      "Bob",
			To:        "Charlie",
			Amount:    25,
			Timestamp: time.Now(),
			PublicKey: "bob_public_key",
			Nonce:     1,
			Hash:      "tx_hash_002",
			Signature: "invalid_signature", // Assinatura inválida para teste
		},
	}

	// Testa a mineração com validação
	block := mineBlockWithValidation(node, pendingTxs)

	fmt.Printf("\n📦 Bloco minerado:")
	fmt.Printf("   Index: %d\n", block.Index)
	fmt.Printf("   Hash: %s\n", block.Hash)
	fmt.Printf("   Transações válidas: %d\n", len(block.Transactions))
	fmt.Printf("   Minerador: %s\n", block.MinerID)

	// Testa validação individual
	fmt.Println("\n🧪 Testando validação individual...")
	validator := NewTransactionValidator()

	// Teste com transação válida
	validTx := pendingTxs[0]
	fmt.Printf("Transação %s: %v\n", validTx.ID, validator.VerifySignature(&validTx))

	// Teste com transação inválida
	invalidTx := pendingTxs[1]
	fmt.Printf("Transação %s: %v\n", invalidTx.ID, validator.VerifySignature(&invalidTx))

	// Teste de recompensa do sistema
	reward, _ := createMiningReward("test_miner", 1)
	fmt.Printf("Recompensa do sistema %s: %v\n", reward.ID, validator.VerifySignature(reward))

	fmt.Println("\n✅ Teste do Secure Miner concluído!")
}
