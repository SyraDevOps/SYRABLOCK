package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Token representa um bloco na blockchain
type Token struct {
	Index        int    `json:"index"`
	Hash         string `json:"hash"`
	PrevHash     string `json:"prev_hash"`
	Timestamp    string `json:"timestamp"`
	Nonce        int    `json:"nonce"`
	MinerID      string `json:"miner_id,omitempty"` // vazio, será preenchido pela carteira depois
	Difficulty   int    `json:"difficulty"`
	CustomToken  string `json:"custom_token"`
	ContainsSyra bool   `json:"contains_syra"`
	Transactions []any  `json:"transactions"`
}

// Stats mantém estatísticas de mineração
type Stats struct {
	startTime  time.Time
	attempts   int
	hashRate   float64
	lastUpdate time.Time
	mu         sync.Mutex
}

func (s *Stats) update(attempts int, customToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts += attempts
	elapsed := time.Since(s.startTime).Seconds()
	s.hashRate = float64(s.attempts) / elapsed

	// Atualiza a exibição a cada segundo
	if time.Since(s.lastUpdate) >= time.Second {
		s.lastUpdate = time.Now()
		fmt.Print("\r                                                                               \r")
		fmt.Printf("⛏️ Hash rate: %.2f H/s | Tentativas: %d | Tempo: %.1fs | Procurando: %s e 'Syra'",
			s.hashRate, s.attempts, elapsed, customToken)
	}
}

// hasLeadingZeros verifica se a string tem o número especificado de zeros no início
func hasLeadingZeros(hash string, zeros int) bool {
	if zeros <= 0 {
		return true // Não exige zeros se zeros <= 0
	}
	prefix := strings.Repeat("0", zeros)
	return strings.HasPrefix(hash, prefix)
}

// saveToken salva o token encontrado em um arquivo JSON
func saveToken(token Token, filename string) error {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(token)
}

// Gera uma string aleatória de n bytes codificada em base64
func randomBase64String(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

var customToken string

func main() {
	difficulty := flag.Int("zeros", 0, "Número de zeros no início do hash (opcional, padrão: 0)")
	prevHash := flag.String("prev", "0000000000000000000000000000000000000000000000000000000000000000", "Hash do bloco anterior")
	flag.StringVar(&customToken, "token", "", "Token personalizado obrigatório (além do Syra)")
	batchSize := flag.Int("batch", 10000, "Tamanho do lote para processamento")
	outputFile := flag.String("output", "special_token.json", "Arquivo para salvar o token encontrado")
	flag.Parse()

	if customToken == "" {
		fmt.Println("❌ Erro: o token personalizado é obrigatório (use --token=\"SEUTOKEN\")")
		return
	}

	token := Token{
		Index:        1,
		PrevHash:     *prevHash,
		Timestamp:    time.Now().Format(time.RFC3339),
		Difficulty:   *difficulty,
		CustomToken:  customToken,
		Transactions: []any{},
	}

	fmt.Printf("🚀 PTW Minerador Especial iniciado\n")
	fmt.Printf("📋 Configurações:\n")
	fmt.Printf("   • Zeros iniciais: %d\n", *difficulty)
	fmt.Printf("   • Token personalizado: '%s'\n", customToken)
	fmt.Printf("   • Hash anterior: %s...\n", (*prevHash)[:16])
	fmt.Printf("   • Requisito fixo: contém 'Syra'\n")
	fmt.Printf("   • Arquivo de saída: %s\n", *outputFile)
	fmt.Printf("\n⏳ Mineração iniciada... (Ctrl+C para cancelar)\n\n")

	stats := &Stats{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}

	// Canal para receber solução e interrupção
	solution := make(chan Token)
	stopMining := make(chan bool)

	// Canal para capturar sinais do sistema (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Inicie mineração em uma goroutine
	go func() {
		nonce := 0

		for {
			select {
			case <-stopMining:
				return
			default:
				for i := 0; i < *batchSize; i++ {
					nonce++

					// Estratégia: gera partes aleatórias, concatena, faz SHA-256 e base64
					partes := []string{
						randomBase64String(8),
						randomBase64String(8),
						randomBase64String(8),
						randomBase64String(8),
					}
					concat := strings.Join(partes, "")
					hashBytes := sha256.Sum256([]byte(concat))
					hashBase64 := base64.StdEncoding.EncodeToString(hashBytes[:])

					// Verifica condições do hash
					if hasLeadingZeros(hashBase64, *difficulty) &&
						strings.Contains(hashBase64, "Syra") &&
						strings.Contains(hashBase64, customToken) {

						currentToken := token
						currentToken.Nonce = nonce
						currentToken.Hash = hashBase64
						currentToken.ContainsSyra = true
						currentToken.Transactions = []any{partes}
						solution <- currentToken
						return
					}
				}
				stats.update(*batchSize, customToken)
			}
		}
	}()

	// Espera por solução ou interrupção
	var foundToken Token
	var found bool

	select {
	case foundToken = <-solution:
		found = true
		close(stopMining)
	case <-sigChan:
		// Interrompido pelo usuário
		fmt.Printf("\n\n🛑 Mineração interrompida pelo usuário após %.1f segundos\n",
			time.Since(stats.startTime).Seconds())
		found = false
		close(stopMining)
	}

	if found {
		elapsed := time.Since(stats.startTime)
		fmt.Printf("\n\n✅ Bloco Encontrado!\n")
		fmt.Printf("⏱️ Tempo total: %s\n", elapsed.Round(time.Millisecond))
		fmt.Printf("🔍 Hash: %s\n", foundToken.Hash)
		fmt.Printf("🔢 Nonce: %d\n", foundToken.Nonce)
		fmt.Printf("📄 PrevHash: %s\n", foundToken.PrevHash)

		// Salva o token
		err := saveToken(foundToken, *outputFile)
		if err != nil {
			fmt.Printf("❌ Erro ao salvar token: %v\n", err)
		} else {
			absPath, _ := filepath.Abs(*outputFile)
			fmt.Printf("💾 Token salvo em: %s\n", absPath)
		}
	}
}
