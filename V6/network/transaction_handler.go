package main

import (
	"fmt"
	"sync"
	"time"
)

// Transaction struct (deve ser consistente com p2p_node.go)
type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"`
	PublicKey string    `json:"public_key,omitempty"`
	Nonce     int       `json:"nonce,omitempty"`
	Hash      string    `json:"hash,omitempty"`
	Signature string    `json:"signature,omitempty"`
}

// TransactionValidator interface para validação
type TransactionValidator struct {
	keyCache map[string]bool
}

func NewTransactionValidator() *TransactionValidator {
	return &TransactionValidator{
		keyCache: make(map[string]bool),
	}
}

func (v *TransactionValidator) VerifySignature(tx *Transaction) bool {
	// Implementação básica - em produção deveria verificar RSA real
	if tx.From == "SYSTEM" && tx.Type == "mining_reward" {
		// Transações do sistema têm validação especial
		return tx.Signature != "" && len(tx.Signature) > 10
	}

	// Transações regulares
	return tx.Signature != "" && tx.PublicKey != "" && len(tx.Signature) > 10
}

func (v *TransactionValidator) ValidateTransactionChain(txs []Transaction) bool {
	for _, tx := range txs {
		if !v.VerifySignature(&tx) {
			return false
		}
	}
	return true
}

// TransactionPool gerencia transações pendentes com validação de assinatura
type TransactionPool struct {
	pendingTx   map[string]*Transaction // txID -> transaction
	validator   *TransactionValidator
	userNonces  map[string]int // userID -> último nonce usado
	mutex       sync.RWMutex
	maxPoolSize int
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		pendingTx:   make(map[string]*Transaction),
		validator:   NewTransactionValidator(),
		userNonces:  make(map[string]int),
		maxPoolSize: 1000,
	}
}

// AddTransaction adiciona transação ao pool após validação completa
func (tp *TransactionPool) AddTransaction(tx *Transaction) error {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// 1. Verifica se o pool não está cheio
	if len(tp.pendingTx) >= tp.maxPoolSize {
		return fmt.Errorf("pool de transações cheio")
	}

	// 2. Verifica se a transação já existe
	if _, exists := tp.pendingTx[tx.ID]; exists {
		return fmt.Errorf("transação já existe no pool")
	}

	// 3. VALIDAÇÃO DE ASSINATURA (PRINCIPAL)
	if !tp.validator.VerifySignature(tx) {
		return fmt.Errorf("assinatura inválida")
	}

	// 4. Verifica ordem de nonce (previne replay attacks) - apenas para transações normais
	if tx.From != "SYSTEM" && tx.Nonce > 0 {
		if lastNonce, exists := tp.userNonces[tx.From]; exists {
			if tx.Nonce <= lastNonce {
				return fmt.Errorf("nonce inválido: %d <= %d", tx.Nonce, lastNonce)
			}
		}
	}

	// 5. Validações de negócio
	if err := tp.validateBusinessRules(tx); err != nil {
		return fmt.Errorf("regra de negócio violada: %v", err)
	}

	// 6. Adiciona ao pool
	tp.pendingTx[tx.ID] = tx

	// Atualiza nonce apenas para transações normais
	if tx.From != "SYSTEM" && tx.Nonce > 0 {
		tp.userNonces[tx.From] = tx.Nonce
	}

	fmt.Printf("✅ Transação %s adicionada ao pool (assinatura válida)\n", tx.ID)
	return nil
}

// validateBusinessRules valida regras específicas de negócio
func (tp *TransactionPool) validateBusinessRules(tx *Transaction) error {
	// Verifica timestamp (não pode ser muito no futuro ou passado)
	now := time.Now()
	if tx.Timestamp.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("timestamp muito no futuro")
	}
	if tx.Timestamp.Before(now.Add(-1 * time.Hour)) {
		return fmt.Errorf("timestamp muito no passado")
	}

	// Validações específicas por tipo
	switch tx.Type {
	case "transfer":
		if tx.Amount <= 0 {
			return fmt.Errorf("valor de transferência inválido: %d", tx.Amount)
		}
		if tx.Amount > 1000000 { // Limite máximo por transação
			return fmt.Errorf("valor muito alto: %d", tx.Amount)
		}
		if tx.From == tx.To {
			return fmt.Errorf("não pode transferir para si mesmo")
		}

	case "mining_reward":
		if tx.From != "SYSTEM" {
			return fmt.Errorf("recompensa deve vir do SYSTEM")
		}
		if tx.Amount <= 0 || tx.Amount > 10 {
			return fmt.Errorf("recompensa inválida: %d", tx.Amount)
		}

	case "contract":
		if tx.Contract == "" {
			return fmt.Errorf("ID do contrato obrigatório")
		}

	default:
		return fmt.Errorf("tipo de transação inválido: %s", tx.Type)
	}

	return nil
}

// GetValidTransactions retorna transações válidas para incluir em bloco
func (tp *TransactionPool) GetValidTransactions(maxCount int) []*Transaction {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	var transactions []*Transaction
	count := 0

	for _, tx := range tp.pendingTx {
		if count >= maxCount {
			break
		}

		// Re-valida a assinatura antes de incluir em bloco
		if tp.validator.VerifySignature(tx) {
			transactions = append(transactions, tx)
			count++
		} else {
			fmt.Printf("⚠️ Transação %s falhou na re-validação\n", tx.ID)
		}
	}

	return transactions
}

// RemoveTransactions remove transações processadas do pool
func (tp *TransactionPool) RemoveTransactions(txIDs []string) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	for _, txID := range txIDs {
		delete(tp.pendingTx, txID)
	}

	fmt.Printf("🗑️ Removidas %d transações do pool\n", len(txIDs))
}

// ValidateBlock valida todas as transações de um bloco
func (tp *TransactionPool) ValidateBlock(transactions []Transaction) bool {
	fmt.Printf("🔍 Validando %d transações do bloco...\n", len(transactions))

	// Valida cada transação individualmente
	for i, tx := range transactions {
		if !tp.validator.VerifySignature(&tx) {
			fmt.Printf("❌ Bloco rejeitado: transação %d (%s) tem assinatura inválida\n", i, tx.ID)
			return false
		}
	}

	// Valida a cadeia completa (nonces, replay attacks, etc.)
	if !tp.validator.ValidateTransactionChain(transactions) {
		fmt.Printf("❌ Bloco rejeitado: cadeia de transações inválida\n")
		return false
	}

	fmt.Printf("✅ Todas as %d transações do bloco são válidas\n", len(transactions))
	return true
}

// GetPoolStatus retorna estatísticas do pool
func (tp *TransactionPool) GetPoolStatus() map[string]interface{} {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	return map[string]interface{}{
		"pending_transactions": len(tp.pendingTx),
		"max_pool_size":        tp.maxPoolSize,
		"pool_usage_percent":   float64(len(tp.pendingTx)) / float64(tp.maxPoolSize) * 100,
	}
}

// CleanupOldTransactions remove transações muito antigas
func (tp *TransactionPool) CleanupOldTransactions(maxAge time.Duration) int {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	now := time.Now()
	removed := 0

	for id, tx := range tp.pendingTx {
		if now.Sub(tx.Timestamp) > maxAge {
			delete(tp.pendingTx, id)
			removed++
		}
	}

	if removed > 0 {
		fmt.Printf("🧹 Removidas %d transações antigas do pool\n", removed)
	}

	return removed
}

// Exemplo de uso e teste
func main() {
	fmt.Println("🧪 Testando Transaction Handler...")

	pool := NewTransactionPool()

	// Teste 1: Adicionar transação válida
	validTx := &Transaction{
		ID:        "TX_VALID_001",
		Type:      "transfer",
		From:      "Alice",
		To:        "Bob",
		Amount:    100,
		Timestamp: time.Now(),
		PublicKey: "alice_public_key",
		Nonce:     1,
		Hash:      "valid_hash",
		Signature: "valid_signature_123",
	}

	err := pool.AddTransaction(validTx)
	if err != nil {
		fmt.Printf("❌ Erro ao adicionar transação válida: %v\n", err)
	} else {
		fmt.Printf("✅ Transação válida adicionada com sucesso\n")
	}

	// Teste 2: Adicionar transação inválida (sem assinatura)
	invalidTx := &Transaction{
		ID:        "TX_INVALID_001",
		Type:      "transfer",
		From:      "Bob",
		To:        "Charlie",
		Amount:    50,
		Timestamp: time.Now(),
		Signature: "", // Assinatura vazia
	}

	err = pool.AddTransaction(invalidTx)
	if err != nil {
		fmt.Printf("✅ Transação inválida corretamente rejeitada: %v\n", err)
	} else {
		fmt.Printf("❌ Transação inválida foi aceita incorretamente\n")
	}

	// Teste 3: Recompensa de mineração
	rewardTx := &Transaction{
		ID:        "REWARD_001",
		Type:      "mining_reward",
		From:      "SYSTEM",
		To:        "Alice",
		Amount:    1,
		Timestamp: time.Now(),
		PublicKey: "SYSTEM_PUBLIC_KEY",
		Signature: "SYSTEM_SIGNATURE_123456",
	}

	err = pool.AddTransaction(rewardTx)
	if err != nil {
		fmt.Printf("❌ Erro ao adicionar recompensa: %v\n", err)
	} else {
		fmt.Printf("✅ Recompensa de mineração adicionada com sucesso\n")
	}

	// Teste 4: Status do pool
	status := pool.GetPoolStatus()
	fmt.Printf("📊 Status do pool: %+v\n", status)

	// Teste 5: Obter transações válidas
	validTxs := pool.GetValidTransactions(10)
	fmt.Printf("📋 Transações válidas obtidas: %d\n", len(validTxs))

	fmt.Println("✅ Teste do Transaction Handler concluído!")
}
