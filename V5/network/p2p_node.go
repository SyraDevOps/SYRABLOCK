package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Estruturas principais
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

type P2PNode struct {
	ID          string           `json:"id"`
	Address     string           `json:"address"`
	Port        int              `json:"port"`
	Peers       map[string]*Peer `json:"peers"`
	Blockchain  []Token          `json:"blockchain"`
	PendingTxs  []Transaction    `json:"pending_transactions"`
	IsValidator bool             `json:"is_validator"`
	Stake       int              `json:"stake"`
	mutex       sync.RWMutex
	listener    net.Listener

	// Bitcoin-style discovery
	addrManager      *AddrManager
	dnsSeeder        *DNSSeeder
	bootstrapManager *BootstrapManager
	dht              *DHTTable
}

type Peer struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"last_seen"`
	IsActive bool      `json:"is_active"`
	Stake    int       `json:"stake"`
	conn     net.Conn  `json:"-"`
}

type NetworkMessage struct {
	Type      string      `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Signature string      `json:"signature"`
}

type BootstrapNode struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

// Constantes
const (
	MSG_PEER_DISCOVERY    = "peer_discovery"
	MSG_NEW_BLOCK         = "new_block"
	MSG_NEW_TRANSACTION   = "new_transaction"
	MSG_CONSENSUS_REQUEST = "consensus_request"
	MSG_CONSENSUS_VOTE    = "consensus_vote"
	MSG_SYNC_REQUEST      = "sync_request"
	MSG_SYNC_RESPONSE     = "sync_response"
	MSG_HEARTBEAT         = "heartbeat"
	MSG_ADDR_REQUEST      = "addr_request"
	MSG_ADDR_RESPONSE     = "addr_response"
	MSG_INTRODUCTION      = "introduction"
	MSG_INTRODUCTION_ACK  = "introduction_ack"
)

var DefaultBootstrapNodes = []BootstrapNode{
	{"127.0.0.1", 8080}, // Local para desenvolvimento
	{"127.0.0.1", 8081},
	{"127.0.0.1", 8082},
}

// Construtor
func NewP2PNode(id, address string, port int) *P2PNode {
	return &P2PNode{
		ID:          id,
		Address:     address,
		Port:        port,
		Peers:       make(map[string]*Peer),
		Blockchain:  []Token{},
		PendingTxs:  []Transaction{},
		IsValidator: false,
		Stake:       0,
	}
}

// Inicialização do nó
func (node *P2PNode) StartNode() error {
	// Inicializa o sistema de gerenciamento de endereços
	dataDir := filepath.Join(".", "ptw_data")
	node.addrManager = NewAddrManager(dataDir)
	node.addrManager.Start()

	// Inicializa o sistema DNS
	node.dnsSeeder = NewDNSSeeder(node.addrManager)

	// Inicializa o gerenciador de bootstrap
	node.bootstrapManager = NewBootstrapManager(node.addrManager, node.dnsSeeder)

	node.dht = NewDHTTable(node.ID, node.Address, node.Port, "./ptw_data")

	// Cria certificados TLS self-signed se não existirem
	node.ensureTLSCertificates()

	// Inicia servidor TCP seguro
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		return fmt.Errorf("erro ao carregar certificados: %v", err)
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Para desenvolvimento local
	}

	listenAddr := net.JoinHostPort(node.Address, fmt.Sprintf("%d", node.Port))
	listener, err := tls.Listen("tcp", listenAddr, config)
	if err != nil {
		return fmt.Errorf("erro ao iniciar listener: %v", err)
	}

	node.listener = listener
	fmt.Printf("🌐 Nó P2P iniciado: %s\n", listenAddr)

	// Carrega blockchain existente
	node.loadBlockchainFromFile()

	// Goroutines para diferentes funções
	go node.handleConnections()
	go node.syncBlockchain()
	go node.heartbeatRoutine()
	go node.consensusRoutine()

	// Iniciar descoberta descentralizada
	go node.BitcoinStyleDiscovery()

	go func() {
		time.Sleep(2 * time.Second)
		node.requestPeerAddresses()
	}()

	return nil
}

// Carrega blockchain do arquivo
func (node *P2PNode) loadBlockchainFromFile() {
	file, err := os.Open("../tokens.json")
	if err != nil {
		fmt.Printf("📚 Arquivo de blockchain não encontrado, iniciando com blockchain vazia\n")
		return
	}
	defer file.Close()

	var tokens []Token
	if err := json.NewDecoder(file).Decode(&tokens); err != nil {
		fmt.Printf("❌ Erro ao decodificar blockchain: %v\n", err)
		return
	}

	node.mutex.Lock()
	node.Blockchain = tokens
	node.mutex.Unlock()

	fmt.Printf("📚 Blockchain carregada: %d blocos\n", len(tokens))
}

// Salva blockchain no arquivo
func (node *P2PNode) saveBlockchainToFile() {
	node.mutex.RLock()
	blockchain := make([]Token, len(node.Blockchain))
	copy(blockchain, node.Blockchain)
	node.mutex.RUnlock()

	file, err := os.Create("../tokens.json")
	if err != nil {
		fmt.Printf("❌ Erro ao salvar blockchain: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(blockchain); err != nil {
		fmt.Printf("❌ Erro ao codificar blockchain: %v\n", err)
	}
}

// Cria certificados TLS básicos se não existirem
func (node *P2PNode) ensureTLSCertificates() {
	// Verifica se os arquivos já existem
	if _, err := os.Stat("server.crt"); err == nil {
		if _, err := os.Stat("server.key"); err == nil {
			return // Certificados já existem
		}
	}

	// Cria certificados básicos para desenvolvimento
	certContent := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJANHFxQFQKL7DMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNVBAMMCWxv
Y2FsaG9zdDAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBQxEjAQBgNV
BAMMCWxvY2FsaG9zdDBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7K2QBH6FKz7Mr
MQiGz9Q9m5JKxQx5z1VZoOzRzUvxJgWQqGQ1LMqHKMiQzv1QgWQqGQ1LMqHKMiQ
zv1QgWQqGQ1LAgMBAAEwDQYJKoZIhvcNAQELBQADQQBGJFvT3QJKNwQ6RJrGJlTQ
JKJl4G1OQJFJEjOJKQJQJFJEQj3QKJKJlTQJKJl4G1OQJFJEjOJKQJQJFJEQj3Q
-----END CERTIFICATE-----`

	keyContent := `-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEAuytmAR+hSs+zKzEI
hs/UPZuSSsUMec9VWaDs0c1L8SYFkKhkNSzKhyjIkM79UIFkKhkNSzKhyjIkM79U
IFkKhkNSwIDAQABAkEAzJgMn5dKJKJlTQJKJl4G1OQJFJEjOJKQJQJFJEQj3QKJ
KJlTQJKJl4G1OQJFJEjOJKQJQJFJEQj3QKJKJlTQwIhAOgT8JKNwQ6RJrGJlTQJK
Jl4G1OQJFJEjOJKQJQJFJEQjAiEA6RJrGJlTQJKJl4G1OQJFJEjOJKQJQJFJEQj3
QKJKJlTQCIQDJKJlTQJKJl4G1OQJFJEjOJKQJQJFJEQj3QKJKJlTQJKJlQ
-----END PRIVATE KEY-----`

	os.WriteFile("server.crt", []byte(certContent), 0644)
	os.WriteFile("server.key", []byte(keyContent), 0600)
}

// Implementação de descoberta estilo Bitcoin
func (node *P2PNode) BitcoinStyleDiscovery() {
	fmt.Println("🌐 Iniciando descoberta estilo Bitcoin...")

	connectCallback := func(ip, port string) bool {
		portInt, _ := strconv.Atoi(port)
		peer := &Peer{
			ID:       fmt.Sprintf("%s:%s", ip, port),
			Address:  ip,
			Port:     portInt,
			IsActive: false,
		}
		return node.connectToPeer(peer) == nil
	}

	// Bootstrap inicial
	node.bootstrapManager.InitialConnection(connectCallback)

	// Descoberta contínua
	for {
		time.Sleep(30 * time.Second)

		// Limpa peers inativos
		node.cleanupInactivePeers()

		// Tenta descobrir novos peers se tiver poucos
		if len(node.Peers) < 5 {
			node.requestPeerAddresses()
		}
	}
}

// Conecta a um peer
func (node *P2PNode) connectToPeer(peer *Peer) error {
	address := net.JoinHostPort(peer.Address, fmt.Sprintf("%d", peer.Port))

	// Verifica se já está conectado
	node.mutex.RLock()
	if existingPeer, exists := node.Peers[peer.ID]; exists && existingPeer.IsActive {
		node.mutex.RUnlock()
		return nil
	}
	node.mutex.RUnlock()

	config := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("erro ao conectar: %v", err)
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// Mensagem de apresentação
	introMsg := &NetworkMessage{
		Type: MSG_INTRODUCTION,
		From: node.ID,
		Data: map[string]interface{}{
			"version":           "PTW/1.0",
			"user_agent":        "PTW-Core:1.0.0",
			"address":           node.Address,
			"port":              node.Port,
			"services":          []string{"FULL_NODE", "MINER", "VALIDATOR"},
			"blockchain_height": len(node.Blockchain),
		},
		Timestamp: time.Now(),
	}

	err = encoder.Encode(introMsg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("erro ao enviar introdução: %v", err)
	}

	var response NetworkMessage
	err = decoder.Decode(&response)
	if err != nil {
		conn.Close()
		return fmt.Errorf("erro ao receber resposta: %v", err)
	}

	if response.Type != MSG_INTRODUCTION_ACK {
		conn.Close()
		return fmt.Errorf("resposta inesperada: %s", response.Type)
	}

	// Processa dados do peer
	if data, ok := response.Data.(map[string]interface{}); ok {
		if peerID, ok := data["peer_id"].(string); ok {
			peer.ID = peerID
		}
	}

	// Adiciona na lista de peers ativos
	peer.IsActive = true
	peer.LastSeen = time.Now()
	peer.conn = conn

	node.mutex.Lock()
	node.Peers[peer.ID] = peer
	node.mutex.Unlock()

	fmt.Printf("✅ Conectado ao peer: %s\n", peer.ID)

	// Inicia rotina para lidar com este peer
	go node.handlePeer(conn, peer.ID)

	// Solicita sincronização imediata
	go node.requestSyncFromPeer(peer.ID)

	return nil
}

// Lida com conexões de entrada
func (node *P2PNode) handleConnections() {
	for {
		conn, err := node.listener.Accept()
		if err != nil {
			fmt.Printf("❌ Erro ao aceitar conexão: %v\n", err)
			continue
		}

		go node.handleIncomingConnection(conn)
	}
}

// Lida com conexão de entrada
func (node *P2PNode) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	// Espera mensagem de introdução
	var introMsg NetworkMessage
	if err := decoder.Decode(&introMsg); err != nil {
		return
	}

	if introMsg.Type != MSG_INTRODUCTION {
		return
	}

	// Processa dados do peer
	peerData, ok := introMsg.Data.(map[string]interface{})
	if !ok {
		return
	}

	peerID := introMsg.From
	address := "unknown"
	port := 0

	if addr, ok := peerData["address"].(string); ok {
		address = addr
	}
	if p, ok := peerData["port"].(float64); ok {
		port = int(p)
	}

	// Cria peer
	peer := &Peer{
		ID:       peerID,
		Address:  address,
		Port:     port,
		IsActive: true,
		LastSeen: time.Now(),
		conn:     conn,
	}

	// Adiciona à lista de peers
	node.mutex.Lock()
	node.Peers[peerID] = peer
	node.mutex.Unlock()

	// Envia ACK
	ackMsg := &NetworkMessage{
		Type: MSG_INTRODUCTION_ACK,
		From: node.ID,
		Data: map[string]interface{}{
			"peer_id": node.ID,
			"address": node.Address,
			"port":    node.Port,
		},
		Timestamp: time.Now(),
	}

	if err := encoder.Encode(ackMsg); err != nil {
		return
	}

	fmt.Printf("✅ Peer conectado: %s\n", peerID)

	// Inicia comunicação com este peer
	node.handlePeer(conn, peerID)
}

// Lida com um peer específico
func (node *P2PNode) handlePeer(conn net.Conn, peerID string) {
	defer func() {
		conn.Close()

		// Remove peer da lista ativa
		node.mutex.Lock()
		if peer, exists := node.Peers[peerID]; exists {
			peer.IsActive = false
		}
		node.mutex.Unlock()

		fmt.Printf("🔌 Peer desconectado: %s\n", peerID)
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var msg NetworkMessage
		if err := decoder.Decode(&msg); err != nil {
			break
		}

		// Atualiza última vez visto
		node.mutex.Lock()
		if peer, exists := node.Peers[peerID]; exists {
			peer.LastSeen = time.Now()
		}
		node.mutex.Unlock()

		// Processa mensagem
		response := node.processMessage(&msg)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				break
			}
		}
	}
}

// Processa mensagens recebidas
func (node *P2PNode) processMessage(msg *NetworkMessage) *NetworkMessage {
	switch msg.Type {
	case MSG_PEER_DISCOVERY:
		return node.handlePeerDiscovery(msg)
	case MSG_NEW_BLOCK:
		return node.handleNewBlock(msg)
	case MSG_NEW_TRANSACTION:
		return node.handleNewTransaction(msg)
	case MSG_CONSENSUS_REQUEST:
		return node.handleConsensusRequest(msg)
	case MSG_SYNC_REQUEST:
		return node.handleSyncRequest(msg)
	case MSG_HEARTBEAT:
		return node.handleHeartbeat(msg)
	case MSG_ADDR_REQUEST:
		return node.handleAddrRequest(msg)
	case MSG_ADDR_RESPONSE:
		return node.handleAddrResponse(msg)
	}
	return nil
}

// === IMPLEMENTAÇÕES DOS HANDLERS ===

func (node *P2PNode) handlePeerDiscovery(msg *NetworkMessage) *NetworkMessage {
	// Retorna lista de peers conhecidos
	peers := make([]map[string]interface{}, 0)

	node.mutex.RLock()
	for _, peer := range node.Peers {
		if peer.IsActive && peer.ID != msg.From {
			peers = append(peers, map[string]interface{}{
				"id":      peer.ID,
				"address": peer.Address,
				"port":    peer.Port,
			})
		}
	}
	node.mutex.RUnlock()

	return &NetworkMessage{
		Type: "peer_list",
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"peers": peers,
		},
		Timestamp: time.Now(),
	}
}

func (node *P2PNode) handleNewBlock(msg *NetworkMessage) *NetworkMessage {
	blockData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	// Converte dados para Token
	blockBytes, _ := json.Marshal(blockData)
	var block Token
	if err := json.Unmarshal(blockBytes, &block); err != nil {
		return nil
	}

	if node.validateAndAddBlock(&block) {
		fmt.Printf("📦 Novo bloco adicionado: %s\n", block.Hash[:16])

		// Propaga para outros peers
		go node.BroadcastToNetwork(MSG_NEW_BLOCK, blockData, msg.From)
	}

	return nil
}

func (node *P2PNode) validateAndAddBlock(block *Token) bool {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	// Verifica se já temos este bloco
	for _, existingBlock := range node.Blockchain {
		if existingBlock.Hash == block.Hash {
			return false
		}
	}

	// Validações básicas
	if block.Hash == "" || !block.ContainsSyra {
		return false
	}

	// Verifica índice sequencial
	expectedIndex := len(node.Blockchain) + 1
	if block.Index != expectedIndex {
		return false
	}

	// Verifica prev_hash
	if len(node.Blockchain) > 0 {
		lastBlock := node.Blockchain[len(node.Blockchain)-1]
		if block.PrevHash != lastBlock.Hash {
			return false
		}
	}

	// Adiciona à blockchain
	node.Blockchain = append(node.Blockchain, *block)

	// Salva no arquivo
	go node.saveBlockchainToFile()

	return true
}

func (node *P2PNode) handleNewTransaction(msg *NetworkMessage) *NetworkMessage {
	txData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	// Converte para Transaction
	txBytes, _ := json.Marshal(txData)
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return nil
	}

	if node.validateTransaction(&tx) {
		node.mutex.Lock()
		node.PendingTxs = append(node.PendingTxs, tx)
		node.mutex.Unlock()

		fmt.Printf("💳 Nova transação recebida: %s\n", tx.ID)

		// Propaga para outros peers
		go node.BroadcastToNetwork(MSG_NEW_TRANSACTION, txData, msg.From)
	}

	return nil
}

func (node *P2PNode) validateTransaction(tx *Transaction) bool {
	// Validações básicas
	if tx.From == "" || tx.To == "" || tx.Amount <= 0 {
		return false
	}

	// Verifica se não é duplicada
	node.mutex.RLock()
	for _, pendingTx := range node.PendingTxs {
		if pendingTx.ID == tx.ID {
			node.mutex.RUnlock()
			return false
		}
	}
	node.mutex.RUnlock()

	return true
}

func (node *P2PNode) handleConsensusRequest(msg *NetworkMessage) *NetworkMessage {
	consensusData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	// Valida proposta de consenso
	vote := node.validateConsensusProposal(consensusData)

	return &NetworkMessage{
		Type: MSG_CONSENSUS_VOTE,
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"vote":  vote,
			"voter": node.ID,
			"stake": node.Stake,
		},
		Timestamp: time.Now(),
	}
}

func (node *P2PNode) validateConsensusProposal(consensusData map[string]interface{}) bool {
	// Implementa validação de consenso
	// Por simplicidade, aprova se tiver stake suficiente
	return node.IsValidator && node.Stake >= 10
}

func (node *P2PNode) handleSyncRequest(msg *NetworkMessage) *NetworkMessage {
	node.mutex.RLock()
	blockchain := make([]Token, len(node.Blockchain))
	copy(blockchain, node.Blockchain)
	node.mutex.RUnlock()

	return &NetworkMessage{
		Type: MSG_SYNC_RESPONSE,
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"blockchain": blockchain,
			"height":     len(blockchain),
		},
		Timestamp: time.Now(),
	}
}

func (node *P2PNode) handleHeartbeat(msg *NetworkMessage) *NetworkMessage {
	// Atualiza peer como ativo
	return &NetworkMessage{
		Type: "heartbeat_ack",
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"blockchain_height": len(node.Blockchain),
			"pending_txs":       len(node.PendingTxs),
		},
		Timestamp: time.Now(),
	}
}

func (node *P2PNode) handleAddrRequest(msg *NetworkMessage) *NetworkMessage {
	addrs := make([]map[string]interface{}, 0)

	// Adiciona peers do addr manager
	if node.addrManager != nil {
		knownAddrs := node.addrManager.GetAddresses(20, true, true)
		for _, addr := range knownAddrs {
			addrs = append(addrs, map[string]interface{}{
				"ip":   addr.IP,
				"port": addr.Port,
			})
		}
	}

	// Adiciona peers ativos
	node.mutex.RLock()
	for _, peer := range node.Peers {
		if peer.IsActive && peer.ID != msg.From {
			addrs = append(addrs, map[string]interface{}{
				"ip":   peer.Address,
				"port": fmt.Sprintf("%d", peer.Port),
			})
		}
	}
	node.mutex.RUnlock()

	return &NetworkMessage{
		Type: MSG_ADDR_RESPONSE,
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"addresses": addrs,
		},
		Timestamp: time.Now(),
	}
}

func (node *P2PNode) handleAddrResponse(msg *NetworkMessage) *NetworkMessage {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	addresses, ok := data["addresses"].([]interface{})
	if !ok {
		return nil
	}

	// Adiciona endereços ao addr manager
	for _, addrInterface := range addresses {
		if addrMap, ok := addrInterface.(map[string]interface{}); ok {
			if ip, ok := addrMap["ip"].(string); ok {
				if port, ok := addrMap["port"].(string); ok {
					if node.addrManager != nil {
						node.addrManager.AddAddress(ip, port, "peer")
					}
				}
			}
		}
	}

	return nil
}

// === MÉTODOS AUXILIARES ===

func (node *P2PNode) requestPeerAddresses() {
	node.mutex.RLock()
	peers := make([]*Peer, 0)
	for _, peer := range node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	node.mutex.RUnlock()

	for _, peer := range peers {
		msg := &NetworkMessage{
			Type: MSG_ADDR_REQUEST,
			From: node.ID,
			To:   peer.ID,
			Data: map[string]interface{}{
				"max_addresses": 50,
			},
			Timestamp: time.Now(),
		}

		go node.sendToPeer(peer, msg)
	}
}

func (node *P2PNode) requestSyncFromPeer(peerID string) {
	node.mutex.RLock()
	peer, exists := node.Peers[peerID]
	node.mutex.RUnlock()

	if !exists || !peer.IsActive {
		return
	}

	msg := &NetworkMessage{
		Type: MSG_SYNC_REQUEST,
		From: node.ID,
		To:   peerID,
		Data: map[string]interface{}{
			"request_type": "full_blockchain",
			"my_height":    len(node.Blockchain),
		},
		Timestamp: time.Now(),
	}

	node.sendToPeer(peer, msg)
}

// === ROTINAS EM BACKGROUND ===

func (node *P2PNode) syncBlockchain() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		node.requestBlockchainSync()
	}
}

func (node *P2PNode) requestBlockchainSync() {
	node.mutex.RLock()
	peers := make([]*Peer, 0)
	for _, peer := range node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	myHeight := len(node.Blockchain)
	node.mutex.RUnlock()

	for _, peer := range peers {
		msg := &NetworkMessage{
			Type: MSG_SYNC_REQUEST,
			From: node.ID,
			To:   peer.ID,
			Data: map[string]interface{}{
				"my_height": myHeight,
			},
			Timestamp: time.Now(),
		}

		go node.sendToPeer(peer, msg)
	}
}

func (node *P2PNode) heartbeatRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		node.sendHeartbeats()
		node.cleanupInactivePeers()
	}
}

func (node *P2PNode) sendHeartbeats() {
	node.mutex.RLock()
	peers := make([]*Peer, 0)
	for _, peer := range node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	node.mutex.RUnlock()

	for _, peer := range peers {
		msg := &NetworkMessage{
			Type: MSG_HEARTBEAT,
			From: node.ID,
			To:   peer.ID,
			Data: map[string]interface{}{
				"timestamp": time.Now().Unix(),
			},
			Timestamp: time.Now(),
		}

		go node.sendToPeer(peer, msg)
	}
}

func (node *P2PNode) cleanupInactivePeers() {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	now := time.Now()
	for id, peer := range node.Peers {
		if peer.IsActive && now.Sub(peer.LastSeen) > 2*time.Minute {
			peer.IsActive = false
			if peer.conn != nil {
				peer.conn.Close()
			}
			fmt.Printf("🔌 Peer inativo removido: %s\n", id)
		}
	}
}

func (node *P2PNode) consensusRoutine() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		node.initiateConsensus()
	}
}

func (node *P2PNode) initiateConsensus() {
	// Implementa início de consenso com transações pendentes
	// Por simplicidade, apenas limpa transações antigas
	node.mutex.Lock()
	now := time.Now()
	validTxs := make([]Transaction, 0)

	for _, tx := range node.PendingTxs {
		if now.Sub(tx.Timestamp) < 10*time.Minute {
			validTxs = append(validTxs, tx)
		}
	}

	node.PendingTxs = validTxs
	node.mutex.Unlock()
}

func (node *P2PNode) BroadcastToNetwork(msgType string, data interface{}, excludePeer string) {
	node.mutex.RLock()
	peers := make([]*Peer, 0)
	for _, peer := range node.Peers {
		if peer.IsActive && peer.ID != excludePeer {
			peers = append(peers, peer)
		}
	}
	node.mutex.RUnlock()

	for _, peer := range peers {
		msg := &NetworkMessage{
			Type:      msgType,
			From:      node.ID,
			To:        peer.ID,
			Data:      data,
			Timestamp: time.Now(),
		}

		go node.sendToPeer(peer, msg)
	}
}

func (node *P2PNode) sendToPeer(peer *Peer, msg *NetworkMessage) {
	if peer.conn == nil {
		return
	}

	encoder := json.NewEncoder(peer.conn)
	if err := encoder.Encode(msg); err != nil {
		peer.IsActive = false
	}
}

// === DESCOBERTA DE PEERS ===

func (node *P2PNode) DiscoverPeers() error {
	// Tenta conectar aos nós bootstrap
	for _, bootstrap := range DefaultBootstrapNodes {
		if bootstrap.Port != node.Port { // Não conecta a si mesmo
			go node.connectToBootstrap(bootstrap)
		}
	}

	// Descoberta multicast local
	go node.localMulticastDiscovery()

	return nil
}

func (node *P2PNode) connectToBootstrap(bootstrap BootstrapNode) {
	peer := &Peer{
		ID:      fmt.Sprintf("%s:%d", bootstrap.Address, bootstrap.Port),
		Address: bootstrap.Address,
		Port:    bootstrap.Port,
	}

	if err := node.connectToPeer(peer); err != nil {
		fmt.Printf("❌ Falha ao conectar ao bootstrap %s:%d: %v\n",
			bootstrap.Address, bootstrap.Port, err)
	}
}

func (node *P2PNode) localMulticastDiscovery() {
	// Implementa descoberta multicast local
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.1:9999")
	if err != nil {
		return
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	// Envia anúncio
	announcement := fmt.Sprintf("PTW_NODE:%s:%d", node.Address, node.Port)
	conn.WriteTo([]byte(announcement), addr)

	// Escuta anúncios
	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			continue
		}

		message := string(buffer[:n])
		if strings.HasPrefix(message, "PTW_NODE:") {
			parts := strings.Split(message, ":")
			if len(parts) == 3 {
				peerAddr := parts[1]
				peerPort, _ := strconv.Atoi(parts[2])

				if peerPort != node.Port { // Não conecta a si mesmo
					peer := &Peer{
						ID:      fmt.Sprintf("%s:%d", peerAddr, peerPort),
						Address: peerAddr,
						Port:    peerPort,
					}
					go node.connectToPeer(peer)
				}
			}
		}
	}
}

// === MÉTODOS PARA BLOCKCHAIN ===

func (node *P2PNode) StartConsensusRound(block interface{}) {
	// Implementação de consenso distribuído
	if tokenBlock, ok := block.(*Token); ok {
		fmt.Printf("🗳️ Iniciando consenso para bloco: %s\n", tokenBlock.Hash[:16])

		consensusData := map[string]interface{}{
			"block_hash": tokenBlock.Hash,
			"block":      tokenBlock,
			"proposer":   node.ID,
		}

		node.BroadcastToNetwork(MSG_CONSENSUS_REQUEST, consensusData, "")
	}
}

// Função main para teste (pode ser removida se não precisar)
func main() {
	if len(os.Args) < 4 {
		fmt.Println("Uso: go run p2p_node.go <node_id> <address> <port>")
		return
	}

	nodeID := os.Args[1]
	address := os.Args[2]
	port, _ := strconv.Atoi(os.Args[3])

	node := NewP2PNode(nodeID, address, port)
	if err := node.StartNode(); err != nil {
		fmt.Printf("Erro ao iniciar nó: %v\n", err)
		return
	}

	// Mantém o programa rodando
	select {}
}
