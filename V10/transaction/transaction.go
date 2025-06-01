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
		fmt.Printf("❌ Transação %s: hash alterado\n", tx.ID)
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
		fmt.Printf("❌ Transação %s: verificação de assinatura falhou\n", tx.ID)
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
		if tx.From != "SYSTEM" && tx.Nonce > 0 {
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
			fmt.Printf("❌ Valor de transferência inválido: %d\n", tx.Amount)
			return false
		}
		if tx.From == tx.To {
			fmt.Printf("❌ Não pode transferir para si mesmo\n")
			return false
		}
	case "mining_reward":
		if tx.From != "SYSTEM" {
			fmt.Printf("❌ Recompensa de mineração deve vir do SYSTEM\n")
			return false
		}
		if tx.Amount <= 0 {
			fmt.Printf("❌ Recompensa deve ser positiva\n")
			return false
		}
	case "contract":
		if tx.Contract == "" {
			fmt.Printf("❌ Transação de contrato deve especificar o contrato\n")
			return false
		}
	default:
		fmt.Printf("❌ Tipo de transação desconhecido: %s\n", tx.Type)
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
	fmt.Println("🧪 Testando Sistema de Transações...")

	validator := NewTransactionValidator()

	fmt.Println("\n=== Teste 1: Transação Válida ===")
	tx1, err := CreateTransaction("Alice", "Bob", 100, "transfer", "")
	if err != nil {
		fmt.Printf("Erro ao criar transação: %v\n", err)
	} else {
		fmt.Printf("Transação criada: %s\n", tx1.ID)
		valid := validator.VerifySignature(tx1)
		fmt.Printf("Válida: %v\n", valid)
	}

	fmt.Println("\n=== Teste 2: Validação de Cadeia ===")
	transactions := []Transaction{}
	if tx1 != nil {
		transactions = append(transactions, *tx1)
	}

	validChain := validator.ValidateTransactionChain(transactions)
	fmt.Printf("Cadeia válida: %v\n", validChain)

	fmt.Println("\n=== Teste 3: Transação com Hash Alterado ===")
	if tx1 != nil {
		invalidTx := *tx1
		invalidTx.Amount = 200 // Altera valor mas mantém assinatura
		invalidTx.Hash, _ = invalidTx.calculateHash()

		valid := validator.VerifySignature(&invalidTx)
		fmt.Printf("Transação com hash alterado válida: %v\n", valid)
	}

	fmt.Println("\n✅ Testes de transação concluídos!")
}

func testTransactionValidation() {
	validator := NewTransactionValidator()

	// Cria transação de teste válida
	tx := &Transaction{
		ID:        "TEST_TX_001",
		Type:      "transfer",
		From:      "Alice",
		To:        "Bob",
		Amount:    50,
		Timestamp: time.Now(),
		PublicKey: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END PUBLIC KEY-----",
		Signature: "valid_signature_here",
		Hash:      "valid_hash_here",
		Nonce:     1,
	}

	valid := validator.VerifySignature(tx)
	fmt.Printf("Transação válida: %v\n", valid)

	// Testa transação com assinatura inválida
	invalidTx := *tx
	invalidTx.Signature = "invalid_signature"

	valid = validator.VerifySignature(&invalidTx)
	fmt.Printf("Transação com assinatura inválida: %v\n", valid)

	// Testa transação com hash alterado
	alteredTx := *tx
	alteredTx.Amount = 200 // Altera valor mas mantém assinatura

	valid = validator.VerifySignature(&alteredTx)
	fmt.Printf("Transação com hash alterado: %v\n", valid)
}
