package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "transfer", "contract", "mining_reward"
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"`
	PublicKey string    `json:"public_key"` // Chave pública do remetente
	Signature string    `json:"signature"`  // Assinatura digital da transação
	Hash      string    `json:"hash"`       // Hash da transação (para integridade)
	Nonce     int       `json:"nonce"`      // Previne replay attacks
}

type KeyPair struct {
	UserID     string `json:"user_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	CreatedAt  string `json:"created_at"`
}

// TransactionValidator valida assinaturas de transações
type TransactionValidator struct {
	keyCache map[string]*rsa.PublicKey // Cache de chaves públicas
}

func NewTransactionValidator() *TransactionValidator {
	return &TransactionValidator{
		keyCache: make(map[string]*rsa.PublicKey),
	}
}

// CreateTransaction cria uma nova transação assinada
func CreateTransaction(fromID, toID string, amount int, txType string, privateKeyPath string) (*Transaction, error) {
	// Carrega chave privada
	keyPair, err := loadKeyPair(fromID)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar chave privada: %v", err)
	}

	// Cria transação base
	tx := &Transaction{
		ID:        fmt.Sprintf("TX_%d_%s", time.Now().UnixNano(), fromID),
		Type:      txType,
		From:      fromID,
		To:        toID,
		Amount:    amount,
		Timestamp: time.Now(),
		PublicKey: keyPair.PublicKey,
		Nonce:     int(time.Now().UnixNano() % 1000000), // Nonce simples
	}

	// Calcula hash da transação
	txHash, err := tx.calculateHash()
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular hash: %v", err)
	}
	tx.Hash = txHash

	// Assina a transação
	signature, err := tx.signTransaction(keyPair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("erro ao assinar transação: %v", err)
	}
	tx.Signature = signature

	return tx, nil
}

// calculateHash calcula hash SHA256 da transação (sem signature)
func (tx *Transaction) calculateHash() (string, error) {
	// Cria cópia sem signature para o hash
	txCopy := *tx
	txCopy.Signature = ""
	txCopy.Hash = ""

	txBytes, err := json.Marshal(txCopy)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(txBytes)
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// signTransaction assina a transação com a chave privada
func (tx *Transaction) signTransaction(privateKeyPEM string) (string, error) {
	// Decodifica chave privada
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("falha ao decodificar chave privada")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("não é uma chave RSA")
	}

	// Hash da transação para assinar
	hashBytes := sha256.Sum256([]byte(tx.Hash))

	// Assina o hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPrivateKey, crypto.SHA256, hashBytes[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifica se a assinatura da transação é válida
func (tv *TransactionValidator) VerifySignature(tx *Transaction) bool {
	// Valida campos obrigatórios
	if tx.From == "" || tx.PublicKey == "" || tx.Signature == "" || tx.Hash == "" {
		fmt.Printf("❌ Transação %s: campos obrigatórios faltando\n", tx.ID)
		return false
	}

	// Carrega chave pública do cache ou decodifica
	publicKey := tv.getPublicKey(tx.From, tx.PublicKey)
	if publicKey == nil {
		fmt.Printf("❌ Transação %s: chave pública inválida\n", tx.ID)
		return false
	}

	// Verifica integridade do hash
	expectedHash, err := tx.calculateHash()
	if err != nil {
		fmt.Printf("❌ Transação %s: erro ao calcular hash\n", tx.ID)
		return false
	}

	if expectedHash != tx.Hash {
		fmt.Printf("❌ Transação %s: hash não confere\n", tx.ID)
		return false
	}

	// Decodifica assinatura
	signature, err := base64.StdEncoding.DecodeString(tx.Signature)
	if err != nil {
		fmt.Printf("❌ Transação %s: assinatura inválida\n", tx.ID)
		return false
	}

	// Verifica assinatura
	hashBytes := sha256.Sum256([]byte(tx.Hash))
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashBytes[:], signature)
	if err != nil {
		fmt.Printf("❌ Transação %s: assinatura não confere\n", tx.ID)
		return false
	}

	fmt.Printf("✅ Transação %s: assinatura válida\n", tx.ID)
	return true
}

// getPublicKey obtém chave pública do cache ou decodifica
func (tv *TransactionValidator) getPublicKey(userID, publicKeyPEM string) *rsa.PublicKey {
	// Verifica cache primeiro
	if cachedKey, exists := tv.keyCache[userID]; exists {
		return cachedKey
	}

	// Decodifica chave pública
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil
	}

	// Adiciona ao cache
	tv.keyCache[userID] = rsaPublicKey
	return rsaPublicKey
}

// ValidateTransactionChain valida uma cadeia de transações
func (tv *TransactionValidator) ValidateTransactionChain(transactions []Transaction) bool {
	usedNonces := make(map[string]map[int]bool) // userID -> nonce -> usado

	for _, tx := range transactions {
		// Verifica assinatura
		if !tv.VerifySignature(&tx) {
			return false
		}

		// Verifica replay attack (nonce duplicado)
		if usedNonces[tx.From] == nil {
			usedNonces[tx.From] = make(map[int]bool)
		}

		if usedNonces[tx.From][tx.Nonce] {
			fmt.Printf("❌ Replay attack detectado: nonce %d já usado por %s\n", tx.Nonce, tx.From)
			return false
		}

		usedNonces[tx.From][tx.Nonce] = true

		// Validações específicas por tipo
		if !tv.validateTransactionType(&tx) {
			return false
		}
	}

	return true
}

// validateTransactionType valida regras específicas por tipo de transação
func (tv *TransactionValidator) validateTransactionType(tx *Transaction) bool {
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
		if tx.Amount <= 0 || tx.Amount > 10 { // Limite máximo de recompensa
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

// Funções auxiliares
func loadKeyPair(userID string) (*KeyPair, error) {
	filename := filepath.Join("..", "PWtSY", fmt.Sprintf("keypair_%s.json", userID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keyPair KeyPair
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&keyPair)
	return &keyPair, err
}

// Exemplo de uso
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run transaction.go <comando> [parametros]")
		fmt.Println("Comandos:")
		fmt.Println("  create <from> <to> <amount> <type>  - Cria transação assinada")
		fmt.Println("  verify <transaction_file>           - Verifica assinatura")
		fmt.Println("  test                                - Teste de validação")
		return
	}

	switch os.Args[1] {
	case "create":
		if len(os.Args) < 6 {
			fmt.Println("Uso: create <from> <to> <amount> <type>")
			return
		}

		from := os.Args[2]
		to := os.Args[3]
		var amount int
		fmt.Sscanf(os.Args[4], "%d", &amount)
		txType := os.Args[5]

		tx, err := CreateTransaction(from, to, amount, txType, "")
		if err != nil {
			fmt.Printf("Erro ao criar transação: %v\n", err)
			return
		}

		// Salva transação
		filename := fmt.Sprintf("transaction_%s.json", tx.ID)
		file, err := os.Create(filename)
		if err != nil {
			fmt.Printf("Erro ao salvar transação: %v\n", err)
			return
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encoder.Encode(tx)

		fmt.Printf("✅ Transação criada e assinada: %s\n", tx.ID)
		fmt.Printf("   Hash: %s\n", tx.Hash[:16]+"...")
		fmt.Printf("   Arquivo: %s\n", filename)

	case "verify":
		if len(os.Args) < 3 {
			fmt.Println("Uso: verify <transaction_file>")
			return
		}

		filename := os.Args[2]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Erro ao abrir arquivo: %v\n", err)
			return
		}
		defer file.Close()

		var tx Transaction
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&tx); err != nil {
			fmt.Printf("Erro ao decodificar transação: %v\n", err)
			return
		}

		validator := NewTransactionValidator()
		if validator.VerifySignature(&tx) {
			fmt.Println("✅ Assinatura da transação é VÁLIDA")
		} else {
			fmt.Println("❌ Assinatura da transação é INVÁLIDA")
		}

	case "test":
		testTransactionValidation()

	default:
		fmt.Println("Comando não reconhecido")
	}
}

func testTransactionValidation() {
	fmt.Println("🧪 Testando validação de transações...")

	validator := NewTransactionValidator()

	// Cria transação de teste válida
	fmt.Println("\n1. Testando transação válida:")
	validTx, err := CreateTransaction("Alice", "Bob", 50, "transfer", "")
	if err != nil {
		fmt.Printf("Erro ao criar transação: %v\n", err)
		return
	}

	if validator.VerifySignature(validTx) {
		fmt.Println("✅ Transação válida verificada com sucesso")
	} else {
		fmt.Println("❌ Falha na verificação de transação válida")
	}

	// Testa transação com assinatura inválida
	fmt.Println("\n2. Testando transação com assinatura inválida:")
	invalidTx := *validTx
	invalidTx.Signature = "assinatura_invalida"

	if !validator.VerifySignature(&invalidTx) {
		fmt.Println("✅ Transação inválida corretamente rejeitada")
	} else {
		fmt.Println("❌ Transação inválida foi aceita incorretamente")
	}

	// Testa transação com hash alterado
	fmt.Println("\n3. Testando transação com hash alterado:")
	tamperedTx := *validTx
	tamperedTx.Amount = 999 // Altera valor mas mantém assinatura

	if !validator.VerifySignature(&tamperedTx) {
		fmt.Println("✅ Transação adulterada corretamente rejeitada")
	} else {
		fmt.Println("❌ Transação adulterada foi aceita incorretamente")
	}

	fmt.Println("\n🏁 Teste de validação concluído")
}
