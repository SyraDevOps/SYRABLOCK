package main

import (
	"bufio"
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

// hasCustomPrefix verifica se o hash começa com o prefixo personalizado
func hasCustomPrefix(hash, prefix string) bool {
	if prefix == "" {
		return true
	}
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

func getLastHashFromFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	var lastHash string
	for {
		var t Token
		if err := dec.Decode(&t); err != nil {
			break
		}
		lastHash = t.Hash
	}
	if lastHash == "" {
		return "", fmt.Errorf("nenhum hash encontrado no arquivo")
	}
	return lastHash, nil
}

func promptForPrevHash() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Digite o hash prévio para este token: ")
	prevHash, _ := reader.ReadString('\n')
	return strings.TrimSpace(prevHash)
}

var customToken string

func main() {
	difficulty := flag.Int("zeros", 0, "Número de zeros no início do hash (opcional, padrão: 0)")
	customPrefix := flag.String("dific", "", "Prefixo personalizado que o hash deve começar (ex: 27, 004, lin, Sy, ky)")
	prevHash := flag.String("prev", "", "Hash do bloco anterior (opcional, será detectado automaticamente)")
	flag.StringVar(&customToken, "token", "", "Token personalizado obrigatório (além do Syra)")
	batchSize := flag.Int("batch", 10000, "Tamanho do lote para processamento")
	outputFile := flag.String("output", "special_token.json", "Arquivo para salvar o token encontrado")
	flag.Parse()

	if customToken == "" {
		fmt.Println("❌ Erro: o token personalizado é obrigatório (use --token=\"SEUTOKEN\")")
		return
	}

	// NOVO: Detecta hash prévio automaticamente do arquivo de saída, se existir
	prev := *prevHash
	if prev == "" {
		if _, err := os.Stat(*outputFile); err == nil {
			// Arquivo existe, tenta pegar o último hash
			lastHash, err := getLastHashFromFile(*outputFile)
			if err == nil && lastHash != "" {
				prev = lastHash
				fmt.Printf("ℹ️  Usando hash prévio do último token salvo: %s\n", prev)
			}
		}
	}
	if prev == "" {
		// Solicita ao usuário
		prev = promptForPrevHash()
		if prev == "" {
			fmt.Println("❌ Hash prévio é obrigatório para iniciar a mineração.")
			return
		}
	}

	token := Token{
		Index:        1,
		PrevHash:     prev,
		Timestamp:    time.Now().Format(time.RFC3339),
		Difficulty:   *difficulty,
		CustomToken:  customToken,
		Transactions: []any{},
	}

	fmt.Printf("🚀 PTW Minerador Especial iniciado\n")
	fmt.Printf("📋 Configurações:\n")
	if *customPrefix != "" {
		fmt.Printf("   • Prefixo personalizado: '%s'\n", *customPrefix)
	} else {
		fmt.Printf("   • Zeros iniciais: %d\n", *difficulty)
	}
	fmt.Printf("   • Token personalizado: '%s'\n", customToken)
	fmt.Printf("   • Hash anterior: %s...\n", prev[:16])
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

					// NOVO: Verifica prefixo personalizado OU zeros
					if hasCustomPrefix(hashBase64, *customPrefix) &&
						(*customPrefix != "" || hasLeadingZeros(hashBase64, *difficulty)) &&
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
