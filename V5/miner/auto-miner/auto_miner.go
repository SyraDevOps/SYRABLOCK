package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	outputFile = "../tokens.json"
	searchWord = "Syra"
)

type Transaction struct {
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"`
}

type Token struct {
	Index           int           `json:"index"`
	Nonce           int           `json:"nonce"`
	Hash            string        `json:"hash"`
	HashParts       []string      `json:"hash_parts"`
	Timestamp       string        `json:"timestamp"`
	ContainsSyra    bool          `json:"contains_syra"`
	Validator       string        `json:"validator,omitempty"`
	PrevHash        string        `json:"prev_hash,omitempty"`
	WalletAddress   string        `json:"wallet_address,omitempty"`
	WalletSignature string        `json:"wallet_signature,omitempty"`
	MinerID         string        `json:"miner_id,omitempty"`
	Transactions    []Transaction `json:"transactions,omitempty"`
	MinerReward     int           `json:"miner_reward,omitempty"`
}

type Wallet struct {
	UserID           string   `json:"user_id"`
	UniqueToken      string   `json:"unique_token"`
	Signature        string   `json:"signature"`
	ValidationSeq    string   `json:"validation_sequence"`
	Address          string   `json:"address"`
	Balance          int      `json:"balance"`
	RegisteredBlocks []string `json:"registered_blocks"`
	KYCVerified      bool     `json:"kyc_verified"`
}

func loadWallet(userID string) (*Wallet, error) {
	// Corrige o caminho: de miner\auto-miner para PWtSY
	filename := filepath.Join("..", "..", "PWtSY", fmt.Sprintf("wallet_%s.json", userID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wallet Wallet
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&wallet)
	return &wallet, err
}

func saveWallet(wallet *Wallet) error {
	// Corrige o caminho: de miner\auto-miner para PWtSY
	filename := filepath.Join("..", "..", "PWtSY", fmt.Sprintf("wallet_%s.json", wallet.UserID))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(wallet)
}

func randomString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func generateComplexHash(nonce int, minerID string) (string, []string) {
	var combined string
	var parts []string
	for j := 0; j < 4; j++ {
		randomPart := randomString(8)
		input := fmt.Sprintf("%sSYRA2025%s%d", randomPart, minerID, nonce)
		sum := sha256.Sum256([]byte(input))
		hashPart := base64.StdEncoding.EncodeToString(sum[:])
		parts = append(parts, hashPart)
		combined += hashPart
	}
	finalSum := sha256.Sum256([]byte(combined))
	finalHash := base64.StdEncoding.EncodeToString(finalSum[:])
	return finalHash, parts
}

func loadTokens() ([]Token, map[string]struct{}) {
	var tokens []Token
	existing := make(map[string]struct{})
	// Corrige o caminho para tokens.json
	file, err := os.Open("../../tokens.json")
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&tokens)
		for _, t := range tokens {
			existing[t.Hash] = struct{}{}
		}
	}
	return tokens, existing
}

func saveToFile(tokens []Token) {
	// Corrige o caminho para tokens.json
	file, err := os.Create("../../tokens.json")
	if err != nil {
		fmt.Println("Erro ao criar arquivo:", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokens); err != nil {
		fmt.Println("Erro ao salvar JSON:", err)
	}
}

func existsInMap(m map[string]struct{}, key string) bool {
	_, exists := m[key]
	return exists
}

func logAudit(action, userID, details string) {
	// Corrige o caminho para audit.log
	logFile, err := os.OpenFile("../audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] %s | User: %s | Details: %s\n", timestamp, action, userID, details)
	logFile.WriteString(logEntry)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run auto_miner.go <user_id> <wallet_signature>")
		fmt.Println("Exemplo: go run auto_miner.go Faiolhe H9lYInElYCrtFUFudvZIUZkVYmC2TsKCiX5G/N8+KMY=")
		return
	}

	userID := os.Args[1]
	walletSig := os.Args[2]

	// Carrega e valida carteira
	wallet, err := loadWallet(userID)
	if err != nil {
		fmt.Printf("Erro ao carregar carteira: %v\n", err)
		logAudit("MINER_ERROR", userID, "Carteira não encontrada")
		return
	}

	if wallet.Signature != walletSig {
		fmt.Println("Assinatura da carteira inválida!")
		logAudit("MINER_SECURITY_VIOLATION", userID, "Assinatura inválida tentativa de mineração")
		return
	}

	if !wallet.KYCVerified {
		fmt.Println("Usuário não passou no KYC. Não pode minerar.")
		logAudit("MINER_KYC_VIOLATION", userID, "Tentativa de mineração sem KYC")
		return
	}

	fmt.Printf("Minerando para carteira: %s\n", userID)
	fmt.Printf("Endereço: %s\n", wallet.Address)
	fmt.Println("Minerando... (digite 'q' + Enter para parar)")

	stop := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == "q" {
				close(stop)
				return
			}
		}
	}()

	tokens, existing := loadTokens()
	index := len(tokens) + 1
	blocksMinedSession := 0

	logAudit("MINER_START", userID, fmt.Sprintf("Iniciou mineração na sessão %d", time.Now().Unix()))

loop:
	for {
		select {
		case <-stop:
			break loop
		default:
			nonce := 0
			var hash string
			var parts []string

			startTime := time.Now()
			for {
				hash, parts = generateComplexHash(nonce, userID)
				if strings.Contains(hash, searchWord) && !existsInMap(existing, hash) {
					break
				}
				nonce++
			}
			miningTime := time.Since(startTime)

			containsSyra := strings.Contains(hash, searchWord)
			var prevHash string
			if len(tokens) > 0 {
				prevHash = tokens[len(tokens)-1].Hash
			}

			// Recompensa fixa por bloco minerado
			minerReward := 1

			token := Token{
				Index:           index,
				Nonce:           nonce,
				Hash:            hash,
				HashParts:       parts,
				Timestamp:       time.Now().Format(time.RFC3339),
				ContainsSyra:    containsSyra,
				PrevHash:        prevHash,
				Transactions:    []Transaction{},
				MinerID:         userID,
				WalletAddress:   wallet.Address,
				WalletSignature: wallet.Signature,
				MinerReward:     minerReward,
			}

			tokens = append(tokens, token)
			existing[hash] = struct{}{}

			// Atualiza carteira do minerador
			wallet.RegisteredBlocks = append(wallet.RegisteredBlocks, hash)
			wallet.Balance += minerReward
			saveWallet(wallet)

			blocksMinedSession++

			fmt.Printf("✅ Bloco %d | Nonce: %d | Tempo: %v | Recompensa: %d SYRA\n",
				index, nonce, miningTime, minerReward)
			fmt.Printf("   Hash: %s\n", hash)
			if containsSyra {
				fmt.Println("🔥 Contém 'Syra' no hash!")
			}

			saveToFile(tokens)

			logAudit("BLOCK_MINED", userID, fmt.Sprintf("Bloco %d minerado | Hash: %s | Nonce: %d | Tempo: %v",
				index, hash, nonce, miningTime))

			index++
		}
	}

	fmt.Printf("\nMinerador parado. Blocos minerados nesta sessão: %d\n", blocksMinedSession)
	fmt.Printf("Saldo atual: %d SYRA\n", wallet.Balance)

	logAudit("MINER_STOP", userID, fmt.Sprintf("Parou mineração | Blocos minerados: %d | Saldo: %d",
		blocksMinedSession, wallet.Balance))
}
