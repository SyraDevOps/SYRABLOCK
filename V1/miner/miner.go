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
	"strings"
	"time"
)

const (
	outputFile = "../tokens.json"
	searchWord = "Syra"
)

type Token struct {
	Index        int      `json:"index"`
	Nonce        int      `json:"nonce"`
	Hash         string   `json:"hash"`
	HashParts    []string `json:"hash_parts"`
	Timestamp    string   `json:"timestamp"`
	ContainsSyra bool     `json:"contains_syra"`
	Validator    string   `json:"validator,omitempty"`
	PrevHash     string   `json:"prev_hash,omitempty"` // NOVO CAMPO
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

func main() {
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

	fmt.Println("Minerando... (digite 'q' + Enter para parar)")
	tokens, existing := loadTokens()
	index := len(tokens) + 1

loop:
	for {
		select {
		case <-stop:
			break loop
		default:
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
			index++
		}
	}
	fmt.Println("Minerador parado.")
}

func existsInMap(m map[string]struct{}, key string) bool {
	_, exists := m[key]
	return exists
}
