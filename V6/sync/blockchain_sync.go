package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
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

type NetworkMessage struct {
	Type      string      `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

const (
	MSG_SYNC_REQUEST  = "sync_request"
	MSG_SYNC_RESPONSE = "sync_response"
)

type Peer struct {
	ID          string        `json:"id"`
	Address     string        `json:"address"`
	Port        int           `json:"port"`
	IsActive    bool          `json:"is_active"`
	LastSeen    time.Time     `json:"last_seen"`
	BlockHeight int           `json:"block_height"`
	Latency     time.Duration `json:"latency"`
	Reliability float64       `json:"reliability"`
	conn        interface{}   `json:"-"`
}

type P2PNode struct {
	ID         string           `json:"id"`
	Address    string           `json:"address"`
	Port       int              `json:"port"`
	Peers      map[string]*Peer `json:"peers"`
	Blockchain []Token          `json:"blockchain"`
	mutex      sync.RWMutex
}

type SyncManager struct {
	node         *P2PNode
	isSyncing    bool
	lastSyncTime time.Time
	syncInterval time.Duration
	syncMutex    sync.Mutex

	// Estatísticas de sincronização
	syncAttempts     int
	successfulSyncs  int
	failedSyncs      int
	lastSyncDuration time.Duration
}

type SyncResponse struct {
	Peer       *Peer
	Blockchain []Token
	Height     int
	Latency    time.Duration
	Error      error
}

func NewSyncManager(node *P2PNode) *SyncManager {
	return &SyncManager{
		node:         node,
		isSyncing:    false,
		syncInterval: 30 * time.Second,
	}
}

func (sm *SyncManager) StartSyncRoutine() {
	go func() {
		ticker := time.NewTicker(sm.syncInterval)
		defer ticker.Stop()

		for range ticker.C {
			sm.SyncWithNetwork()
		}
	}()

	// Sincronização inicial
	go func() {
		time.Sleep(5 * time.Second) // Aguarda conexões iniciais
		sm.SyncWithNetwork()
	}()
}

func (sm *SyncManager) SyncWithNetwork() {
	sm.syncMutex.Lock()
	if sm.isSyncing {
		sm.syncMutex.Unlock()
		return
	}
	sm.isSyncing = true
	sm.syncMutex.Unlock()

	defer func() {
		sm.syncMutex.Lock()
		sm.isSyncing = false
		sm.lastSyncTime = time.Now()
		sm.syncMutex.Unlock()
	}()

	startTime := time.Now()
	sm.syncAttempts++

	fmt.Println("🔄 Iniciando sincronização da blockchain...")

	// Coleta informações de todos os peers
	responses := sm.requestBlockchainInfo()

	if len(responses) == 0 {
		fmt.Println("❌ Nenhum peer disponível para sincronização")
		sm.failedSyncs++
		return
	}

	// Analisa respostas e encontra a melhor blockchain
	bestChain, bestPeer := sm.findBestChain(responses)

	if bestChain == nil {
		fmt.Println("❌ Nenhuma blockchain válida encontrada")
		sm.failedSyncs++
		return
	}

	currentHeight := len(sm.node.Blockchain)
	newHeight := len(bestChain)

	if newHeight > currentHeight {
		fmt.Printf("📥 Sincronizando: %d -> %d blocos (peer: %s)\n",
			currentHeight, newHeight, bestPeer.ID)

		if sm.applyBlockchain(bestChain) {
			sm.successfulSyncs++
			sm.updatePeerReliability(bestPeer, true)
			fmt.Printf("✅ Sincronização concluída com sucesso\n")
		} else {
			sm.failedSyncs++
			sm.updatePeerReliability(bestPeer, false)
			fmt.Printf("❌ Falha na sincronização\n")
		}
	} else if newHeight == currentHeight {
		fmt.Println("✅ Blockchain já está sincronizada")
		sm.successfulSyncs++
	} else {
		fmt.Printf("📤 Nossa blockchain é mais longa (%d vs %d) - propagando\n",
			currentHeight, newHeight)
		sm.propagateOurBlockchain()
	}

	sm.lastSyncDuration = time.Since(startTime)
}

func (sm *SyncManager) requestBlockchainInfo() []SyncResponse {
	sm.node.mutex.RLock()
	peers := make([]*Peer, 0)
	for _, peer := range sm.node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	sm.node.mutex.RUnlock()

	if len(peers) == 0 {
		return []SyncResponse{}
	}

	// Canal para receber respostas
	responsesChan := make(chan SyncResponse, len(peers))

	// Solicita informações de cada peer em paralelo
	for _, peer := range peers {
		go func(p *Peer) {
			response := sm.requestFromSinglePeer(p)
			responsesChan <- response
		}(peer)
	}

	// Coleta respostas com timeout
	var responses []SyncResponse
	timeout := time.After(10 * time.Second)
	expectedResponses := len(peers)

	for i := 0; i < expectedResponses; i++ {
		select {
		case response := <-responsesChan:
			if response.Error == nil {
				responses = append(responses, response)
			}
		case <-timeout:
			fmt.Printf("⏰ Timeout aguardando resposta de peers\n")
			break
		}
	}

	return responses
}

func (sm *SyncManager) requestFromSinglePeer(peer *Peer) SyncResponse {
	startTime := time.Now()

	request := &NetworkMessage{
		Type: MSG_SYNC_REQUEST,
		From: sm.node.ID,
		To:   peer.ID,
		Data: map[string]interface{}{
			"request_type": "full_blockchain",
			"my_height":    len(sm.node.Blockchain),
		},
		Timestamp: time.Now(),
	}

	response := sm.sendSyncRequest(peer, request)
	latency := time.Since(startTime)

	if response == nil {
		return SyncResponse{
			Peer:    peer,
			Error:   fmt.Errorf("sem resposta"),
			Latency: latency,
		}
	}

	// Processa resposta
	responseData, ok := response.Data.(map[string]interface{})
	if !ok {
		return SyncResponse{
			Peer:    peer,
			Error:   fmt.Errorf("dados inválidos"),
			Latency: latency,
		}
	}

	blockchainData, ok := responseData["blockchain"].([]interface{})
	if !ok {
		return SyncResponse{
			Peer:    peer,
			Error:   fmt.Errorf("blockchain inválida"),
			Latency: latency,
		}
	}

	// Converte para []Token
	blockchain := make([]Token, len(blockchainData))
	for i, blockData := range blockchainData {
		blockBytes, _ := json.Marshal(blockData)
		json.Unmarshal(blockBytes, &blockchain[i])
	}

	height := len(blockchain)
	if heightData, ok := responseData["height"].(float64); ok {
		height = int(heightData)
	}

	// Atualiza informações do peer
	peer.BlockHeight = height
	peer.Latency = latency
	peer.LastSeen = time.Now()

	return SyncResponse{
		Peer:       peer,
		Blockchain: blockchain,
		Height:     height,
		Latency:    latency,
	}
}

func (sm *SyncManager) findBestChain(responses []SyncResponse) ([]Token, *Peer) {
	if len(responses) == 0 {
		return nil, nil
	}

	var bestChain []Token
	var bestPeer *Peer
	maxScore := -1.0

	currentHeight := len(sm.node.Blockchain)

	for _, response := range responses {
		if len(response.Blockchain) <= currentHeight {
			continue // Não é melhor que a nossa
		}

		// Valida a blockchain recebida
		if !sm.validateFullChain(response.Blockchain) {
			fmt.Printf("❌ Blockchain inválida do peer %s\n", response.Peer.ID)
			sm.updatePeerReliability(response.Peer, false)
			continue
		}

		// Calcula pontuação baseada em múltiplos fatores
		score := sm.calculateChainScore(response)

		if score > maxScore {
			maxScore = score
			bestChain = response.Blockchain
			bestPeer = response.Peer
		}
	}

	return bestChain, bestPeer
}

func (sm *SyncManager) calculateChainScore(response SyncResponse) float64 {
	// Pontuação baseada em:
	// 1. Comprimento da blockchain (40%)
	// 2. Confiabilidade do peer (30%)
	// 3. Latência (20%)
	// 4. Última atividade (10%)

	lengthScore := float64(len(response.Blockchain)) * 0.4

	reliabilityScore := response.Peer.Reliability * 30.0

	// Penaliza alta latência
	latencyScore := 20.0
	if response.Latency > 5*time.Second {
		latencyScore = 5.0
	} else if response.Latency > 1*time.Second {
		latencyScore = 10.0
	}

	// Premia peers ativos recentemente
	activityScore := 10.0
	if time.Since(response.Peer.LastSeen) > 5*time.Minute {
		activityScore = 2.0
	}

	totalScore := lengthScore + reliabilityScore + latencyScore + activityScore

	fmt.Printf("📊 Score do peer %s: %.2f (L:%.1f R:%.1f Lt:%.1f A:%.1f)\n",
		response.Peer.ID, totalScore, lengthScore, reliabilityScore,
		latencyScore, activityScore)

	return totalScore
}

func (sm *SyncManager) validateFullChain(chain []Token) bool {
	if len(chain) == 0 {
		return false
	}

	// Valida integridade da cadeia
	for i, block := range chain {
		if !sm.validateBlockDetailed(&block, i) {
			return false
		}

		// Verifica encadeamento
		if i > 0 && block.PrevHash != chain[i-1].Hash {
			fmt.Printf("❌ Erro de encadeamento no bloco %d\n", block.Index)
			return false
		}
	}

	return true
}

func (sm *SyncManager) validateBlockDetailed(block *Token, index int) bool {
	// 1. Índice correto
	if block.Index != index+1 {
		return false
	}

	// 2. Hash não vazio
	if block.Hash == "" {
		return false
	}

	// 3. Contém "Syra"
	if !block.ContainsSyra {
		return false
	}

	// 4. Timestamp válido (não muito no futuro)
	if block.Timestamp != "" {
		if blockTime, err := time.Parse(time.RFC3339, block.Timestamp); err == nil {
			if blockTime.After(time.Now().Add(5 * time.Minute)) {
				return false
			}
		}
	}

	// 5. Valida transações se existirem
	for _, tx := range block.Transactions {
		if !sm.validateTransaction(&tx) {
			return false
		}
	}

	return true
}

func (sm *SyncManager) validateTransaction(tx *Transaction) bool {
	if tx.From == "" || tx.To == "" || tx.Amount <= 0 {
		return false
	}
	return true
}

func (sm *SyncManager) applyBlockchain(newChain []Token) bool {
	// Faz backup da blockchain atual
	sm.node.mutex.RLock()
	oldBlockchain := make([]Token, len(sm.node.Blockchain))
	copy(oldBlockchain, sm.node.Blockchain)
	sm.node.mutex.RUnlock()

	// Aplica nova blockchain
	sm.node.mutex.Lock()
	sm.node.Blockchain = make([]Token, len(newChain))
	copy(sm.node.Blockchain, newChain)
	sm.node.mutex.Unlock()

	// Salva no arquivo
	if err := sm.saveBlockchainToFile(); err != nil {
		// Reverte em caso de erro
		sm.node.mutex.Lock()
		sm.node.Blockchain = oldBlockchain
		sm.node.mutex.Unlock()

		fmt.Printf("❌ Erro ao salvar blockchain: %v\n", err)
		return false
	}

	fmt.Printf("💾 Blockchain salva com %d blocos\n", len(newChain))
	return true
}

func (sm *SyncManager) saveBlockchainToFile() error {
	sm.node.mutex.RLock()
	blockchain := make([]Token, len(sm.node.Blockchain))
	copy(blockchain, sm.node.Blockchain)
	sm.node.mutex.RUnlock()

	file, err := os.Create("../tokens.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(blockchain)
}

func (sm *SyncManager) propagateOurBlockchain() {
	sm.node.mutex.RLock()
	blockchain := make([]Token, len(sm.node.Blockchain))
	copy(blockchain, sm.node.Blockchain)
	peers := make([]*Peer, 0)
	for _, peer := range sm.node.Peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	sm.node.mutex.RUnlock()

	for _, peer := range peers {
		go func(p *Peer) {
			msg := &NetworkMessage{
				Type: MSG_SYNC_RESPONSE,
				From: sm.node.ID,
				To:   p.ID,
				Data: map[string]interface{}{
					"blockchain": blockchain,
					"height":     len(blockchain),
				},
				Timestamp: time.Now(),
			}
			sm.sendSyncRequest(p, msg)
		}(peer)
	}
}

func (sm *SyncManager) updatePeerReliability(peer *Peer, success bool) {
	if success {
		peer.Reliability = (peer.Reliability * 0.9) + 0.1
		if peer.Reliability > 1.0 {
			peer.Reliability = 1.0
		}
	} else {
		peer.Reliability = peer.Reliability * 0.8
		if peer.Reliability < 0.0 {
			peer.Reliability = 0.0
		}
	}
}

func (sm *SyncManager) sendSyncRequest(peer *Peer, request *NetworkMessage) *NetworkMessage {
	if peer == nil || !peer.IsActive {
		return nil
	}

	var conn net.Conn
	var err error

	if peer.conn == nil {
		address := net.JoinHostPort(peer.Address, fmt.Sprintf("%d", peer.Port))
		conn, err = tls.Dial("tcp", address, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return nil
		}
		defer conn.Close()
	} else {
		var ok bool
		conn, ok = peer.conn.(net.Conn)
		if !ok {
			return nil
		}
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := encoder.Encode(request); err != nil {
		return nil
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var response NetworkMessage
	if err := decoder.Decode(&response); err != nil {
		return nil
	}

	return &response
}

// Estatísticas de sincronização
func (sm *SyncManager) GetSyncStats() map[string]interface{} {
	successRate := 0.0
	if sm.syncAttempts > 0 {
		successRate = float64(sm.successfulSyncs) / float64(sm.syncAttempts) * 100
	}

	return map[string]interface{}{
		"sync_attempts":      sm.syncAttempts,
		"successful_syncs":   sm.successfulSyncs,
		"failed_syncs":       sm.failedSyncs,
		"success_rate":       successRate,
		"last_sync_time":     sm.lastSyncTime,
		"last_sync_duration": sm.lastSyncDuration,
		"is_syncing":         sm.isSyncing,
	}
}

// Handler para requisições de sincronização
func (node *P2PNode) handleSyncRequest(msg *NetworkMessage) *NetworkMessage {
	data := msg.Data.(map[string]interface{})

	if requestType, ok := data["request_type"].(string); ok {
		switch requestType {
		case "full_blockchain":
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
	}

	return nil
}
