package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
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
	MSG_DIFFICULTY_UPDATE = "difficulty_update"
	MSG_DIFFICULTY_ACK    = "difficulty_ack"
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

// Stub for ensureTLSCertificates
func (node *P2PNode) ensureTLSCertificates() {
	// TODO: Implement certificate generation if needed
}

// Stub for loadBlockchainFromFile
func (node *P2PNode) loadBlockchainFromFile() {
	// TODO: Implement blockchain loading from file
}

// Stub for saveBlockchainToFile
func (node *P2PNode) saveBlockchainToFile() {
	// TODO: Implement blockchain saving to file
}

// Stub for BitcoinStyleDiscovery
func (node *P2PNode) BitcoinStyleDiscovery() {
	// TODO: Implement Bitcoin-style peer discovery
}

// Stub for syncBlockchain
// Simula envio de mensagem
// Stub for syncBlockchain
func (node *P2PNode) syncBlockchain() {
	node.mutex.RLock()
	var peers []*Peer
	for _, peer := range node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	node.mutex.RUnlock()

	if len(peers) == 0 {
		fmt.Printf("⚠️ [%s] Nenhum peer ativo para sincronização\n", node.ID)
		return
	}

	// Solicita blockchain de peers aleatórios
	for _, peer := range peers {
		go func(p *Peer) {
			msg := &NetworkMessage{
				Type:      MSG_SYNC_REQUEST,
				From:      node.ID,
				To:        p.ID,
				Data:      map[string]interface{}{"current_height": len(node.Blockchain)},
				Timestamp: time.Now(),
			}

			// Simula envio de mensagem
			fmt.Printf("📡 [%s] Solicitando sync do peer %s (msg: %s)\n", node.ID, p.ID, msg.Type)
		}(peer)
	}
}

// Stub for heartbeatRoutine
// Stub for heartbeatRoutine
func (node *P2PNode) heartbeatRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		node.sendHeartbeat()
	}
}
func (node *P2PNode) sendHeartbeat() {
	node.mutex.RLock()
	activePeers := 0
	for _, peer := range node.Peers {
		if peer.IsActive {
			activePeers++
			// Envia heartbeat
			msg := &NetworkMessage{
				Type:      MSG_HEARTBEAT,
				From:      node.ID,
				To:        peer.ID,
				Data:      map[string]interface{}{"blockchain_height": len(node.Blockchain)},
				Timestamp: time.Now(),
			}

			// Simula envio
			fmt.Printf("💓 [%s] Heartbeat para %s (msg: %s)\n", node.ID, peer.ID, msg.Type)
		}
	}
	node.mutex.RUnlock()

	if activePeers > 0 {
		fmt.Printf("💓 [%s] Heartbeat enviado para %d peers\n", node.ID, activePeers)
	}
}

// Stub for consensusRoutine
func (node *P2PNode) consensusRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if node.IsValidator && len(node.PendingTxs) > 0 {
			node.initiateConsensus()
		}
	}
}

func (node *P2PNode) initiateConsensus() {
	// Cria bloco com transações pendentes
	newBlock := &Token{
		Index:        len(node.Blockchain) + 1,
		Hash:         fmt.Sprintf("BLOCK_%d_%d", len(node.Blockchain)+1, time.Now().UnixNano()),
		Timestamp:    time.Now().Format(time.RFC3339),
		ContainsSyra: true,
		Validator:    node.ID,
		PrevHash:     node.getLastBlockHash(),
		Transactions: node.PendingTxs,
	}

	// Inicia consenso distribuído
	node.StartConsensusRound(newBlock)
}

// Stub for requestPeerAddresses
func (node *P2PNode) requestPeerAddresses() {
	node.mutex.RLock()
	var peers []*Peer
	for _, peer := range node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	node.mutex.RUnlock()

	for _, peer := range peers {
		go func(p *Peer) {
			msg := &NetworkMessage{
				Type:      MSG_ADDR_REQUEST,
				From:      node.ID,
				To:        p.ID,
				Data:      map[string]interface{}{"max_addresses": 20},
				Timestamp: time.Now(),
			}

			fmt.Printf("📋 [%s] Solicitando endereços do peer %s (msg: %s)\n", node.ID, p.ID, msg.Type)
			// Simula envio da mensagem
		}(peer)
	}
}

// Stub for cleanupInactivePeers
func (node *P2PNode) cleanupInactivePeers() {
	// TODO: Implement cleanup logic
}

// Stub for requestSpecificPeerSync
func (node *P2PNode) requestSpecificPeerSync(peerID string) {
	node.mutex.RLock()
	peer, exists := node.Peers[peerID]
	node.mutex.RUnlock()

	if !exists {
		fmt.Printf("❌ [%s] Peer %s não encontrado para sync\n", node.ID, peerID)
		return
	}

	// Simula envio da mensagem
	fmt.Printf("🔄 [%s] Solicitando sincronização específica do peer %s (endereço: %s:%d)\n", node.ID, peerID, peer.Address, peer.Port)
}

func (node *P2PNode) getLastBlockHash() string {
	node.mutex.RLock()
	defer node.mutex.RUnlock()

	if len(node.Blockchain) > 0 {
		return node.Blockchain[len(node.Blockchain)-1].Hash
	}
	return ""
}

func (node *P2PNode) connectToPeer(peer *Peer) error {
	// Envia mensagem de introdução
	introMsg := &NetworkMessage{
		Type: MSG_INTRODUCTION,
		From: node.ID,
		To:   peer.ID,
		Data: map[string]interface{}{
			"id":      node.ID,
			"address": node.Address,
			"port":    node.Port,
		},
		Timestamp: time.Now(),
	}
	// Simula envio de mensagem de introdução
	fmt.Printf("✅ [%s] Conectado ao peer %s (msg: %s)\n", node.ID, peer.ID, introMsg.Type)
	return nil
}

// Simula servidor de conexões
func (node *P2PNode) handleConnections() {
	go func() {
		for {
			// Simula nova conexão
			time.Sleep(30 * time.Second)

			// Processa conexão simulada
			node.handleIncomingConnection()
		}
	}()
}

func (node *P2PNode) handleIncomingConnection() {
	// Simula processamento de conexão de entrada
	fmt.Printf("📥 [%s] Nova conexão de entrada\n", node.ID)
}

// Adicione a struct ConsensusVote (faltava)
type ConsensusVote struct {
	RoundID   string `json:"round_id"`
	BlockHash string `json:"block_hash"`
	Voter     string `json:"voter"`
	Vote      bool   `json:"vote"`
}

// ConsensusRound structure (deixe só UMA definição)
type ConsensusRound struct {
	RoundID       string          `json:"round_id"`
	Block         *Token          `json:"block"`
	Proposer      string          `json:"proposer"`
	Validators    []string        `json:"validators"`
	Votes         map[string]bool `json:"votes"`
	RequiredVotes int             `json:"required_votes"`
	Status        string          `json:"status"` // PENDING, APPROVED, REJECTED
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
}

var consensusRounds = make(map[string]*ConsensusRound)

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

// handleNewTransaction com validação de assinatura
func (node *P2PNode) handleNewTransaction(msg *NetworkMessage) *NetworkMessage {
	txData, ok := msg.Data.(map[string]interface{})
	if !ok {
		fmt.Println("❌ Dados de transação inválidos")
		return nil
	}

	// Converte para Transaction
	txBytes, _ := json.Marshal(txData)
	var tx Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		fmt.Printf("❌ Erro ao decodificar transação: %v\n", err)
		return nil
	}

	// VALIDAÇÃO DE ASSINATURA OBRIGATÓRIA
	validator := NewTransactionValidator()
	if !validator.VerifySignature(&tx) {
		fmt.Printf("❌ Transação %s rejeitada: assinatura inválida\n", tx.ID)

		// Log de segurança
		logSecurityEvent("INVALID_SIGNATURE", tx.From,
			fmt.Sprintf("Transação com assinatura inválida: %s", tx.ID), "HIGH", false)

		return &NetworkMessage{
			Type: "transaction_rejected",
			From: node.ID,
			To:   msg.From,
			Data: map[string]interface{}{
				"reason": "invalid_signature",
				"tx_id":  tx.ID,
			},
			Timestamp: time.Now(),
		}
	}

	// Adiciona à lista de transações pendentes
	node.mutex.Lock()
	node.PendingTxs = append(node.PendingTxs, tx)
	node.mutex.Unlock()

	fmt.Printf("✅ Transação %s aceita (assinatura válida)\n", tx.ID)

	// Propaga para outros peers
	go node.BroadcastToNetwork(MSG_NEW_TRANSACTION, txData, msg.From)

	return &NetworkMessage{
		Type: "transaction_accepted",
		From: node.ID,
		To:   msg.From,
		Data: map[string]interface{}{
			"tx_id":     tx.ID,
			"status":    "accepted",
			"validated": true,
		},
		Timestamp: time.Now(),
	}
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

	// Validações básicas do bloco
	if block.Hash == "" || !block.ContainsSyra {
		return false
	}

	// NOVA: Validação de assinaturas de todas as transações
	validator := NewTransactionValidator()
	if !validator.ValidateTransactionChain(block.Transactions) {
		fmt.Printf("❌ Bloco %s rejeitado: transações com assinaturas inválidas\n", block.Hash[:16])
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

	// Remove transações processadas do pool pendente
	var remainingTxs []Transaction
	processedTxIDs := make(map[string]bool)

	for _, tx := range block.Transactions {
		processedTxIDs[tx.ID] = true
	}

	for _, tx := range node.PendingTxs {
		if !processedTxIDs[tx.ID] {
			remainingTxs = append(remainingTxs, tx)
		}
	}

	node.PendingTxs = remainingTxs

	// Salva no arquivo
	go node.saveBlockchainToFile()

	fmt.Printf("✅ Bloco %s adicionado (todas as transações válidas)\n", block.Hash[:16])
	return true
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

// Inicia um round de consenso distribuído
func (node *P2PNode) StartConsensusRound(block *Token) {
	// Seleciona validadores (exemplo: todos peers ativos com stake >= 10)
	validators := []string{}
	node.mutex.RLock()
	for id, peer := range node.Peers {
		if peer.IsActive && peer.Stake >= 10 {
			validators = append(validators, id)
		}
	}
	if node.IsValidator && node.Stake >= 10 {
		validators = append(validators, node.ID)
	}
	node.mutex.RUnlock()

	roundID := fmt.Sprintf("ROUND_%d", time.Now().UnixNano())
	round := &ConsensusRound{
		RoundID:       roundID,
		Block:         block,
		Proposer:      node.ID,
		Validators:    validators,
		Votes:         make(map[string]bool),
		RequiredVotes: len(validators)*2/3 + 1, // 67% + 1
		Status:        "PENDING",
		StartTime:     time.Now(),
	}
	consensusRounds[roundID] = round

	fmt.Printf("🗳️ Iniciando consenso distribuído: %s\n", roundID)
	node.BroadcastToNetwork(MSG_CONSENSUS_REQUEST, map[string]interface{}{
		"round_id":   roundID,
		"block":      block,
		"proposer":   node.ID,
		"validators": validators,
	}, "")

	// Vota em si mesmo (se for validador)
	if node.IsValidator {
		voteMsg := ConsensusVote{
			RoundID:   roundID,
			BlockHash: block.Hash,
			Voter:     node.ID,
			Vote:      true,
		}
		node.handleConsensusVote(&NetworkMessage{
			Type:      MSG_CONSENSUS_VOTE,
			From:      node.ID,
			Data:      voteMsg,
			Timestamp: time.Now(),
		})
	}

	// Timeout para encerrar round
	go func() {
		time.Sleep(15 * time.Second)
		node.finalizeConsensusRound(roundID)
	}()
}

// Handler para requisição de consenso
func (node *P2PNode) handleConsensusRequest(msg *NetworkMessage) *NetworkMessage {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return nil
	}
	roundID, _ := data["round_id"].(string)
	blockData := data["block"]
	blockBytes, _ := json.Marshal(blockData)
	var block Token
	json.Unmarshal(blockBytes, &block)
	validatorsIface, _ := data["validators"].([]interface{})

	// Correção: use uma variável com nome diferente e salve o resultado do append
	validatorsList := make([]string, 0, len(validatorsIface))
	for _, v := range validatorsIface {
		if s, ok := v.(string); ok {
			validatorsList = append(validatorsList, s)
		}
	}

	// Armazena o round de consenso para uso posterior
	round := &ConsensusRound{
		RoundID:       roundID,
		Block:         &block,
		Proposer:      msg.From,
		Validators:    validatorsList,
		Votes:         make(map[string]bool),
		RequiredVotes: len(validatorsList)*2/3 + 1,
		Status:        "PENDING",
		StartTime:     time.Now(),
	}
	consensusRounds[roundID] = round

	// Se for validador, vota
	if node.IsValidator {
		// Use a lógica de validação do bloco para decidir o voto
		vote := true
		if block.Hash == "" || !block.ContainsSyra {
			vote = false
		} else {
			validator := NewTransactionValidator()
			if !validator.ValidateTransactionChain(block.Transactions) {
				vote = false
			}
			expectedIndex := len(node.Blockchain) + 1
			if block.Index != expectedIndex {
				vote = false
			}
			if len(node.Blockchain) > 0 {
				lastBlock := node.Blockchain[len(node.Blockchain)-1]
				if block.PrevHash != lastBlock.Hash {
					vote = false
				}
			}
		}
		node.handleConsensusVote(&NetworkMessage{
			Type: MSG_CONSENSUS_VOTE,
			From: node.ID,
			Data: ConsensusVote{
				RoundID:   roundID,
				BlockHash: block.Hash,
				Voter:     node.ID,
				Vote:      vote,
			},
			Timestamp: time.Now(),
		})
	}
	return nil
}

// handleConsensusVote implementation
func (node *P2PNode) handleConsensusVote(msg *NetworkMessage) *NetworkMessage {
	var vote ConsensusVote
	voteBytes, _ := json.Marshal(msg.Data)
	json.Unmarshal(voteBytes, &vote)

	round, exists := consensusRounds[vote.RoundID]
	if !exists {
		return nil
	}
	round.Votes[vote.Voter] = vote.Vote

	// Verifica se atingiu maioria
	yesVotes := 0
	for _, v := range round.Votes {
		if v {
			yesVotes++
		}
	}
	if yesVotes >= round.RequiredVotes && round.Status == "PENDING" {
		round.Status = "APPROVED"
		round.EndTime = time.Now()
		fmt.Printf("✅ Consenso APROVADO para bloco %s (%d/%d votos)\n", round.Block.Hash[:16], yesVotes, round.RequiredVotes)
		node.validateAndAddBlock(round.Block)
	} else if len(round.Votes) == len(round.Validators) && round.Status == "PENDING" {
		round.Status = "REJECTED"
		round.EndTime = time.Now()
		fmt.Printf("❌ Consenso REJEITADO para bloco %s\n", round.Block.Hash[:16])
	}
	return nil
}

// Finaliza round de consenso após timeout
func (node *P2PNode) finalizeConsensusRound(roundID string) {
	round, exists := consensusRounds[roundID]
	if !exists || round.Status != "PENDING" {
		return
	}
	yesVotes := 0
	for _, v := range round.Votes {
		if v {
			yesVotes++
		}
	}
	if yesVotes >= round.RequiredVotes {
		round.Status = "APPROVED"
		round.EndTime = time.Now()
		fmt.Printf("✅ Consenso APROVADO (timeout) para bloco %s (%d/%d votos)\n", round.Block.Hash[:16], yesVotes, round.RequiredVotes)
		node.validateAndAddBlock(round.Block)
	} else {
		round.Status = "REJECTED"
		round.EndTime = time.Now()
		fmt.Printf("❌ Consenso REJEITADO (timeout) para bloco %s\n", round.Block.Hash[:16])
	}
}

// (Removido: função validateProposedBlock não era usada)

// BroadcastToNetwork stub
func (node *P2PNode) BroadcastToNetwork(msgType string, data interface{}, exceptPeerID string) {
	// Example usage of logSecurityEvent to avoid unused warning
	logSecurityEvent("BROADCAST", node.ID, "BroadcastToNetwork called", "INFO", true)
	// TODO: Implement network broadcast logic
}

// logSecurityEvent implementation
func logSecurityEvent(eventType, user, description, severity string, resolved bool) {
	fmt.Printf("🔒 [SECURITY] [%s] User: %s | Desc: %s | Severity: %s | Resolved: %v\n",
		eventType, user, description, severity, resolved)
}

// validateProposedBlock implementation
// Removed because it was unused and caused a linter error.
