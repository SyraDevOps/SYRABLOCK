package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64" // troquei hex por base64
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	maxTokens  = 100
	outputFile = "tokens.json"
	searchWord = "Syra"
)

type Transaction struct {
	Type      string    `json:"type"` // "transfer" ou "contract"
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"` // ID do contrato, se aplicável
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
	Transactions    []Transaction `json:"transactions,omitempty"` // NOVO
}

func randomString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func generateComplexHash(nonce int) (string, []string) {
	var combined string
	var parts []string
	for j := 0; j < 4; j++ {
		randomPart := randomString(8)
		input := fmt.Sprintf("%sSYRA2025", randomPart)
		sum := sha256.Sum256([]byte(input))
		hashPart := base64.StdEncoding.EncodeToString(sum[:]) // base64 aqui
		parts = append(parts, hashPart)
		combined += hashPart
	}
	finalSum := sha256.Sum256([]byte(combined))
	finalHash := base64.StdEncoding.EncodeToString(finalSum[:]) // base64 aqui
	return finalHash, parts
}

func loadTokens() ([]Token, map[string]struct{}) {
	var tokens []Token
	existing := make(map[string]struct{})
	file, err := os.Open(outputFile)
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
	file, err := os.Create(outputFile)
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

func askContinue() bool {
	fmt.Print("Deseja continuar minerando? (s/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		resp := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return resp == "s" || resp == "sim"
	}
	return false
}

func existsInMap(m map[string]struct{}, key string) bool {
	_, exists := m[key]
	return exists
}

func checkIntegrity(tokens []Token) bool {
	for i := 1; i < len(tokens); i++ {
		if tokens[i].PrevHash != tokens[i-1].Hash {
			fmt.Printf("Integridade quebrada no bloco %d!\n", tokens[i].Index)
			return false
		}
	}
	return true
}

func main() {
	tokens, existing := loadTokens()
	if !checkIntegrity(tokens) {
		fmt.Println("A cadeia de blocos está corrompida!")
		return
	}
	index := len(tokens) + 1

	for index <= maxTokens {
		nonce := 0
		var hash string
		var parts []string
		for {
			hash, parts = generateComplexHash(nonce)
			if strings.Contains(hash, searchWord) && !existsInMap(existing, hash) {
				break
			}
			nonce++
		}

		containsSyra := strings.Contains(hash, searchWord)
		var prevHash string
		if len(tokens) > 0 {
			prevHash = tokens[len(tokens)-1].Hash
		}
		token := Token{
			Index:        index,
			Nonce:        nonce,
			Hash:         hash,
			HashParts:    parts,
			Timestamp:    time.Now().Format(time.RFC3339),
			ContainsSyra: containsSyra,
			PrevHash:     prevHash, // NOVO
		}

		tokens = append(tokens, token)
		existing[hash] = struct{}{}
		fmt.Printf("✅ Token %d | Nonce: %d | Hash: %s\n", index, nonce, hash)
		if containsSyra {
			fmt.Println("🔥 Contém 'Syra' no hash!")
		}
		saveToFile(tokens)

		if !askContinue() {
			break
		}
		index++
	}

	checkIntegrity(tokens)
}
