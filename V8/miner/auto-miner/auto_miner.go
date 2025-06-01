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
	Difficulty      int           `json:"difficulty,omitempty"`  // NOVO: Dificuldade do bloco
	MiningTime      float64       `json:"mining_time,omitempty"` // NOVO: Tempo de mineração
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

// DifficultyManager (versão simplificada para integração)
type DifficultyManager struct {
	CurrentDifficulty int `json:"current_difficulty"`
}

func loadDifficultyManager() *DifficultyManager {
	file, err := os.Open("../../difficulty_config.json")
	if err != nil {
		// Dificuldade padrão se não existir configuração
		return &DifficultyManager{CurrentDifficulty: 4}
	}
	defer file.Close()

	var dm DifficultyManager
	json.NewDecoder(file).Decode(&dm)
	return &dm
}

func (dm *DifficultyManager) GetDifficultyTarget() string {
	target := ""
	for i := 0; i < dm.CurrentDifficulty; i++ {
		target += "0"
	}
	return target
}

func (dm *DifficultyManager) IsValidHash(hash string) bool {
	target := dm.GetDifficultyTarget()
	return len(hash) >= len(target) && hash[:len(target)] == target
}

func loadWallet(userID string) (*Wallet, error) {
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

// ATUALIZADO: generateComplexHash agora usa dificuldade dinâmica
func generateComplexHashWithDifficulty(nonce int, minerID string, difficulty int) (string, []string) {
	var combined string
	var parts []string

	// Gera partes do hash
	for j := 0; j < 4; j++ {
		randomPart := randomString(8)
		input := fmt.Sprintf("%sSYRA2025%s%d%d", randomPart, minerID, nonce, difficulty)
		sum := sha256.Sum256([]byte(input))
		hashPart := base64.StdEncoding.EncodeToString(sum[:])
		parts = append(parts, hashPart)
		combined += hashPart
	}

	// Hash final com componente de dificuldade
	finalInput := fmt.Sprintf("%s%d", combined, difficulty)
	finalSum := sha256.Sum256([]byte(finalInput))
	finalHash := base64.StdEncoding.EncodeToString(finalSum[:])

	return finalHash, parts
}

func loadTokens() ([]Token, map[string]struct{}) {
	var tokens []Token
	existing := make(map[string]struct{})
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
	logFile, err := os.OpenFile("../audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] %s | User: %s | Details: %s\n", timestamp, action, userID, details)
	logFile.WriteString(logEntry)
}

// NOVO: Atualiza dificuldade após mineração
func updateDifficultyAfterBlock(blockIndex int, miningTime time.Duration) {
	// Simula atualizaçãoda dificuldade
	// Em implementação completa, isso seria feito pelo DifficultyManager
	logAudit("DIFFICULTY_UPDATE", "SYSTEM",
		fmt.Sprintf("Bloco %d minerado em %.2fs", blockIndex, miningTime.Seconds()))
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

	// NOVO: Carrega gerenciador de dificuldade
	difficultyManager := loadDifficultyManager()
	fmt.Printf("🎯 Dificuldade atual: %d (target: %s)\n",
		difficultyManager.CurrentDifficulty, difficultyManager.GetDifficultyTarget())

	fmt.Printf("Minerando para carteira: %s\n", userID)
	fmt.Printf("Endereço: %s\n", wallet.Address)
	fmt.Println("Minerando com dificuldade dinâmica... (digite 'q' + Enter para parar)")

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

	logAudit("MINER_START", userID, fmt.Sprintf("Iniciou mineração com dificuldade %d",
		difficultyManager.CurrentDifficulty))

loop:
	for {
		select {
		case <-stop:
			break loop
		default:
			// ATUALIZADO: Recarrega dificuldade a cada bloco
			difficultyManager = loadDifficultyManager()
			currentDifficulty := difficultyManager.CurrentDifficulty

			nonce := 0
			var hash string
			var parts []string
			attempts := 0

			startTime := time.Now()

			// ATUALIZADO: Loop de mineração com dificuldade dinâmica
			for {
				hash, parts = generateComplexHashWithDifficulty(nonce, userID, currentDifficulty)
				attempts++

				// Verifica se atende aos critérios: Syra + Dificuldade + Único
				if strings.Contains(hash, searchWord) &&
					difficultyManager.IsValidHash(hash) &&
					!existsInMap(existing, hash) {
					break
				}

				nonce++

				// Feedback de progresso a cada 50k tentativas
				if attempts%50000 == 0 {
					fmt.Printf("⛏️ Tentativas: %dk | Dificuldade: %d | Target: %s\n",
						attempts/1000, currentDifficulty, difficultyManager.GetDifficultyTarget())
				}
			}
			miningTime := time.Since(startTime)

			containsSyra := strings.Contains(hash, searchWord)
			var prevHash string
			if len(tokens) > 0 {
				prevHash = tokens[len(tokens)-1].Hash
			}

			// Recompensa ajustada pela dificuldade
			minerReward := 1 + (currentDifficulty / 2) // Recompensa extra por dificuldade maior

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
				Difficulty:      currentDifficulty,    // NOVO: Registra dificuldade
				MiningTime:      miningTime.Seconds(), // NOVO: Registra tempo de mineração
			}

			tokens = append(tokens, token)
			existing[hash] = struct{}{}

			// Atualiza carteira do minerador
			wallet.RegisteredBlocks = append(wallet.RegisteredBlocks, hash)
			wallet.Balance += minerReward
			saveWallet(wallet)

			blocksMinedSession++

			// ATUALIZADO: Feedback detalhado com dificuldade
			fmt.Printf("✅ Bloco %d | Nonce: %d | Dificuldade: %d | Tempo: %v | Tentativas: %d | Recompensa: %d SYRA\n",
				index, nonce, currentDifficulty, miningTime, attempts, minerReward)
			fmt.Printf("   Hash: %s\n", hash)
			fmt.Printf("   Target cumprido: %s ✅\n", difficultyManager.GetDifficultyTarget())

			if containsSyra {
				fmt.Println("🔥 Contém 'Syra' no hash!")
			}

			saveToFile(tokens)

			// NOVO: Atualiza sistema de dificuldade
			updateDifficultyAfterBlock(index, miningTime)

			logAudit("BLOCK_MINED", userID, fmt.Sprintf("Bloco %d | Hash: %s | Nonce: %d | Dificuldade: %d | Tempo: %v | Tentativas: %d",
				index, hash, nonce, currentDifficulty, miningTime, attempts))

			index++
		}
	}

	fmt.Printf("\nMinerador parado. Blocos minerados nesta sessão: %d\n", blocksMinedSession)
	fmt.Printf("Saldo atual: %d SYRA\n", wallet.Balance)

	logAudit("MINER_STOP", userID, fmt.Sprintf("Parou mineração | Blocos minerados: %d | Saldo: %d",
		blocksMinedSession, wallet.Balance))
}
