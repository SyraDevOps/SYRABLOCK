package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Type definitions (consolidated and corrected)
type P2PNode struct {
	ID          string
	Address     string
	Port        int
	IsValidator bool
	Stake       int
	Peers       map[string]*Peer
	Blockchain  []*Block
	PendingTxs  []string
	mutex       sync.Mutex
}

type Block struct {
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
}

type Peer struct {
	Address  string
	Port     int
	IsActive bool
}

type Wallet struct {
	UserID           string    `json:"user_id"`
	UniqueToken      string    `json:"unique_token"`
	Signature        string    `json:"signature"`
	ValidationSeq    string    `json:"validation_sequence"`
	CreationDate     time.Time `json:"creation_date"`
	Address          string    `json:"address"`
	Balance          int       `json:"balance"`
	RegisteredBlocks []string  `json:"registered_blocks"`
	KYCVerified      bool      `json:"kyc_verified"`
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
}

type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"`
}

type SyncManager struct {
	node *P2PNode
}

// Constructor functions
func NewP2PNode(id, address string, port int) *P2PNode {
	return &P2PNode{
		ID:         id,
		Address:    address,
		Port:       port,
		Peers:      make(map[string]*Peer),
		Blockchain: []*Block{},
		PendingTxs: []string{},
	}
}

func NewSyncManager(node *P2PNode) *SyncManager {
	return &SyncManager{node: node}
}

// P2PNode methods
func (n *P2PNode) StartNode() error {
	fmt.Printf("🌐 Nó P2P iniciado: %s:%d\n", n.Address, n.Port)
	return nil
}

func (n *P2PNode) DiscoverPeers() {
	fmt.Println("🔍 Descobrindo peers na rede...")
	// Add actual peer discovery logic here
}

func (n *P2PNode) StartConsensusRound(block *Block) {
	fmt.Println("🗳️ Iniciando consenso distribuído...")
	// Add actual consensus logic here
}

func (n *P2PNode) requestBlockchainSync() {
	fmt.Println("📡 Solicitando sincronização da blockchain...")
	// Add actual sync request logic here
}

// SyncManager methods
func (sm *SyncManager) SyncWithNetwork() {
	fmt.Println("🔄 Sincronizando com a rede...")
	sm.node.requestBlockchainSync()
}

// Utility functions
func tokenToBlock(token *Token) *Block {
	if token == nil {
		return nil
	}
	return &Block{
		Index:           token.Index,
		Nonce:           token.Nonce,
		Hash:            token.Hash,
		HashParts:       token.HashParts,
		Timestamp:       token.Timestamp,
		ContainsSyra:    token.ContainsSyra,
		Validator:       token.Validator,
		PrevHash:        token.PrevHash,
		WalletAddress:   token.WalletAddress,
		WalletSignature: token.WalletSignature,
		MinerID:         token.MinerID,
		Transactions:    token.Transactions,
	}
}

func generateComplexHash(nonce int) (string, []string) {
	var combined string
	var parts []string

	for j := 0; j < 4; j++ {
		b := make([]byte, 8)
		rand.Read(b)
		randomPart := fmt.Sprintf("%x", b)
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

func loadBlockchain() []Token {
	var tokens []Token
	file, err := os.Open("../tokens.json")
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&tokens)
	}
	return tokens
}

func saveBlockchain(tokens []Token) {
	file, err := os.Create("../tokens.json")
	if err != nil {
		fmt.Printf("Erro ao salvar blockchain: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(tokens)
}

func loadWallet(userID string) (*Wallet, error) {
	walletPath := fmt.Sprintf("../PWtSY/wallet_%s.json", userID)
	file, err := os.Open(walletPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wallet Wallet
	err = json.NewDecoder(file).Decode(&wallet)
	return &wallet, err
}

func saveWallet(wallet *Wallet) error {
	walletPath := fmt.Sprintf("../PWtSY/wallet_%s.json", wallet.UserID)
	file, err := os.Create(walletPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(wallet)
}

func mineNewBlock(node *P2PNode, index int, tokens []Token) *Token {
	const searchWord = "Syra"

	nonce := 0
	var hash string
	var parts []string

	fmt.Printf("⛏️ Minerando bloco %d...\n", index)

	for {
		hash, parts = generateComplexHash(nonce)
		if strings.Contains(hash, searchWord) {
			break
		}
		nonce++

		// Feedback a cada 100k tentativas
		if nonce%100000 == 0 {
			fmt.Printf("   Tentativa: %d\n", nonce)
		}
	}

	var prevHash string
	if len(tokens) > 0 {
		prevHash = tokens[len(tokens)-1].Hash
	}

	return &Token{
		Index:        index,
		Nonce:        nonce,
		Hash:         hash,
		HashParts:    parts,
		Timestamp:    time.Now().Format(time.RFC3339),
		ContainsSyra: strings.Contains(hash, searchWord),
		PrevHash:     prevHash,
		MinerID:      node.ID,
		Transactions: []Transaction{},
	}
}

func startMining(node *P2PNode, wallet *Wallet) {
	fmt.Printf("⛏️ Iniciando mineração para %s...\n", wallet.UserID)

	// Carrega blockchain atual
	tokens := loadBlockchain()
	index := len(tokens) + 1

	// Minera um bloco
	block := mineNewBlock(node, index, tokens)
	if block != nil {
		fmt.Printf("✅ Bloco minerado: %s\n", block.Hash[:16]+"...")

		// Salva na blockchain local
		tokens = append(tokens, *block)
		saveBlockchain(tokens)

		// Adiciona à blockchain do nó
		node.mutex.Lock()
		blocks := make([]*Block, len(tokens))
		for i := range tokens {
			blocks[i] = tokenToBlock(&tokens[i])
		}
		node.Blockchain = blocks
		node.mutex.Unlock()

		// Inicia consenso
		node.StartConsensusRound(tokenToBlock(block))

		// Atualiza saldo da carteira
		wallet.Balance++
		wallet.RegisteredBlocks = append(wallet.RegisteredBlocks, block.Hash)
		saveWallet(wallet)

		fmt.Printf("💰 Saldo atualizado: %d SYRA\n", wallet.Balance)
	}
}

func startValidator(node *P2PNode, wallet *Wallet) {
	fmt.Printf("🛡️ Ativando modo validador para %s...\n", wallet.UserID)
	node.IsValidator = true
	node.Stake = wallet.Balance
	fmt.Printf("   Stake: %d SYRA\n", node.Stake)
	fmt.Println("✅ Modo validador ativo")
}

func showNodeStatus(node *P2PNode) {
	fmt.Printf("\n📊 Status do Nó: %s\n", node.ID)
	fmt.Printf("   Endereço: %s:%d\n", node.Address, node.Port)
	fmt.Printf("   Peers: %d\n", len(node.Peers))
	fmt.Printf("   Blockchain: %d blocos\n", len(node.Blockchain))
	fmt.Printf("   Transações pendentes: %d\n", len(node.PendingTxs))
	fmt.Printf("   Validador: %v\n", node.IsValidator)
	fmt.Printf("   Stake: %d SYRA\n", node.Stake)
}

func handleInteractiveCommand(node *P2PNode, command string) {
	switch command {
	case "peers":
		fmt.Printf("📡 Peers conectados (%d):\n", len(node.Peers))
		for id, peer := range node.Peers {
			status := "🔴 Inativo"
			if peer.IsActive {
				status = "🟢 Ativo"
			}
			fmt.Printf("  %s - %s:%d %s\n", id, peer.Address, peer.Port, status)
		}

	case "mine":
		fmt.Println("⛏️ Iniciando mineração...")
		wallet, err := loadWallet(node.ID)
		if err != nil {
			fmt.Printf("Erro ao carregar carteira: %v\n", err)
			return
		}
		startMining(node, wallet)

	case "sync":
		fmt.Println("🔄 Sincronizando blockchain...")
		syncManager := NewSyncManager(node)
		syncManager.SyncWithNetwork()

	case "status":
		showNodeStatus(node)

	default:
		fmt.Println("❓ Comando não reconhecido")
	}
}

func startP2PNode(node *P2PNode) {
	fmt.Printf("🚀 Iniciando nó P2P: %s\n", node.ID)

	// Inicia o nó
	if err := node.StartNode(); err != nil {
		fmt.Printf("Erro ao iniciar nó: %v\n", err)
		return
	}

	// Descobre peers
	node.DiscoverPeers()

	// Interface interativa
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\n💬 Comandos disponíveis:")
	fmt.Println("  peers    - Lista peers conectados")
	fmt.Println("  mine     - Minerar bloco")
	fmt.Println("  sync     - Sincronizar blockchain")
	fmt.Println("  status   - Status do nó")
	fmt.Println("  quit     - Sair")

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		handleInteractiveCommand(node, command)

		if command == "quit" {
			break
		}
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Uso: go run p2p_client.go <user_id> <port> <comando>")
		fmt.Println("Comandos:")
		fmt.Println("  start     - Inicia nó P2P")
		fmt.Println("  mine      - Inicia mineração")
		fmt.Println("  validate  - Ativa modo validador")
		fmt.Println("  status    - Status do nó")
		return
	}

	userID := os.Args[1]
	port, _ := strconv.Atoi(os.Args[2])
	command := os.Args[3]

	// Cria nó P2P real
	node := NewP2PNode(userID, "0.0.0.0", port)

	// Carrega carteira do usuário
	wallet, err := loadWallet(userID)
	if err != nil {
		fmt.Printf("Erro ao carregar carteira: %v\n", err)
		return
	}

	// Configura stake se for validador
	if wallet.Balance >= 10 {
		node.IsValidator = true
		node.Stake = wallet.Balance
	}

	switch command {
	case "start":
		startP2PNode(node)
	case "mine":
		startMining(node, wallet)
	case "validate":
		startValidator(node, wallet)
	case "status":
		showNodeStatus(node)
	default:
		fmt.Println("Comando não reconhecido")
	}
}
