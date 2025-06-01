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
	start := time.Now()

	// Cria timeout para a requisição
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	// Simula requisição ao peer
	fmt.Printf("📡 Solicitando blockchain do peer %s:%d\n", peer.Address, peer.Port)

	// Simula resposta do peer
	response := SyncResponse{
		Peer:    peer,
		Height:  len(sm.node.Blockchain) + 5, // Simula blockchain maior
		Latency: time.Since(start),
	}

	// Simula blockchain recebida
	if response.Height > len(sm.node.Blockchain) {
		response.Blockchain = make([]Token, response.Height)
		for i := 0; i < response.Height; i++ {
			response.Blockchain[i] = Token{
				Index:        i + 1,
				Hash:         fmt.Sprintf("SYNC_BLOCK_%d_%s", i+1, peer.ID),
				Timestamp:    time.Now().Add(-time.Duration(response.Height-i) * time.Minute).Format(time.RFC3339),
				ContainsSyra: true,
				MinerID:      peer.ID,
			}
		}
	}

	// Atualiza informações do peer
	peer.BlockHeight = response.Height
	peer.LastSeen = time.Now()
	peer.Latency = response.Latency

	return response
}

func (sm *SyncManager) findBestChain(responses []SyncResponse) ([]Token, *Peer) {
	if len(responses) == 0 {
		return nil, nil
	}

	var bestResponse *SyncResponse
	bestScore := 0.0

	for i := range responses {
		response := &responses[i]
		if response.Error != nil {
			continue
		}

		score := sm.calculateChainScore(*response)
		if score > bestScore {
			bestScore = score
			bestResponse = response
		}
	}

	if bestResponse == nil {
		return nil, nil
	}

	fmt.Printf("🏆 Melhor cadeia encontrada: peer %s (score: %.2f, altura: %d)\n",
		bestResponse.Peer.ID, bestScore, bestResponse.Height)

	return bestResponse.Blockchain, bestResponse.Peer
}

func (sm *SyncManager) calculateChainScore(response SyncResponse) float64 {
	// Pontuação baseada em:
	// 1. Comprimento da blockchain (40%)
	// 2. Confiabilidade do peer (30%)
	// 3. Latência (20%)
	// 4. Última atividade (10%)

	lengthScore := float64(response.Height) * 0.4

	// Calcula score de confiabilidade (simulado)
	reliabilityScore := response.Peer.Reliability * 0.3
	if reliabilityScore == 0 {
		reliabilityScore = 0.5 * 0.3 // Score padrão
	}

	// Penaliza alta latência
	latencyScore := 0.0
	if response.Latency < time.Second {
		latencyScore = 0.2
	} else if response.Latency < 3*time.Second {
		latencyScore = 0.1
	}

	// Premia peers ativos recentemente
	activityScore := 0.0
	if time.Since(response.Peer.LastSeen) < time.Minute {
		activityScore = 0.1
	} else if time.Since(response.Peer.LastSeen) < 5*time.Minute {
		activityScore = 0.05
	}

	totalScore := lengthScore + reliabilityScore + latencyScore + activityScore
	return totalScore
}

func (sm *SyncManager) validateFullChain(chain []Token) bool {
	if len(chain) == 0 {
		return false
	}

	fmt.Printf("🔍 Validando cadeia com %d blocos...\n", len(chain))

	// Valida cada bloco individualmente
	for i, block := range chain {
		if !sm.validateBlockDetailed(&block, i) {
			fmt.Printf("❌ Bloco %d falhou na validação\n", i+1)
			return false
		}
	}

	// Valida integridade da cadeia
	for i := 1; i < len(chain); i++ {
		if chain[i].PrevHash != chain[i-1].Hash {
			fmt.Printf("❌ Integridade quebrada entre blocos %d e %d\n", i, i+1)
			return false
		}
	}

	fmt.Printf("✅ Cadeia validada com sucesso\n")
	return true
}

func (sm *SyncManager) validateBlockDetailed(block *Token, index int) bool {
	// 1. Índice correto
	if block.Index != index+1 {
		fmt.Printf("❌ Índice incorreto: esperado %d, encontrado %d\n", index+1, block.Index)
		return false
	}

	// 2. Hash não vazio
	if block.Hash == "" {
		fmt.Printf("❌ Hash vazio no bloco %d\n", block.Index)
		return false
	}

	// 3. Contém "Syra"
	if !block.ContainsSyra {
		fmt.Printf("❌ Bloco %d não contém 'Syra'\n", block.Index)
		return false
	}

	// 4. Timestamp válido (não muito no futuro)
	if blockTime, err := time.Parse(time.RFC3339, block.Timestamp); err == nil {
		if blockTime.After(time.Now().Add(5 * time.Minute)) {
			fmt.Printf("❌ Timestamp do bloco %d muito no futuro\n", block.Index)
			return false
		}
	}

	// 5. Valida transações se existirem
	for i, tx := range block.Transactions {
		if !sm.validateTransaction(&tx) {
			fmt.Printf("❌ Transação %d do bloco %d inválida\n", i, block.Index)
			return false
		}
	}

	return true
}

func (sm *SyncManager) validateTransaction(tx *Transaction) bool {
	// Validações básicas da transação
	if tx.ID == "" {
		return false
	}

	if tx.Type != "transfer" && tx.Type != "mining_reward" && tx.Type != "contract" {
		return false
	}

	if tx.Amount < 0 {
		return false
	}

	if tx.Type == "transfer" && tx.From == tx.To {
		return false
	}

	return true
}

func (sm *SyncManager) applyBlockchain(newChain []Token) bool {
	// Faz backup da blockchain atual
	backup := make([]Token, len(sm.node.Blockchain))
	copy(backup, sm.node.Blockchain)

	// Aplica nova blockchain
	sm.node.mutex.Lock()
	sm.node.Blockchain = make([]Token, len(newChain))
	copy(sm.node.Blockchain, newChain)
	sm.node.mutex.Unlock()

	// Salva no arquivo
	if err := sm.saveBlockchainToFile(); err != nil {
		// Restaura backup em caso de erro
		sm.node.mutex.Lock()
		sm.node.Blockchain = backup
		sm.node.mutex.Unlock()

		fmt.Printf("❌ Erro ao salvar blockchain: %v\n", err)
		return false
	}

	fmt.Printf("✅ Nova blockchain aplicada com %d blocos\n", len(newChain))
	return true
}

func (sm *SyncManager) saveBlockchainToFile() error {
	sm.node.mutex.RLock()
	defer sm.node.mutex.RUnlock()

	file, err := os.Create("../tokens.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sm.node.Blockchain)
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
