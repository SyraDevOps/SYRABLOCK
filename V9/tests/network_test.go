package tests

import (
	"testing"
	"time"
)

// Estruturas de teste para rede P2P
type TestP2PNode struct {
    ID         string
    Address    string
    Port       int
    Peers      map[string]*TestP2PPeer
    Blockchain []*TestBlock
}

type TestP2PPeer struct {
    ID       string
    Address  string
    Port     int
    IsActive bool
    LastSeen time.Time
}

type TestNetworkMessage struct {
    Type      string
    From      string
    To        string
    Data      interface{}
    Timestamp time.Time
}

// Mock do sistema de rede P2P
type mockP2PSystem struct {
    nodes           map[string]*TestP2PNode
    messages        []TestNetworkMessage
    disconnections  int
    syncAttempts    int
    successfulSyncs int
}

func newMockP2PSystem() *mockP2PSystem {
    return &mockP2PSystem{
        nodes:    make(map[string]*TestP2PNode),
        messages: make([]TestNetworkMessage, 0),
    }
}

func (ps *mockP2PSystem) createNode(id, address string, port int) *TestP2PNode {
    node := &TestP2PNode{
        ID:         id,
        Address:    address,
        Port:       port,
        Peers:      make(map[string]*TestP2PPeer),
        Blockchain: make([]*TestBlock, 0),
    }
    ps.nodes[id] = node
    return node
}

func (ps *mockP2PSystem) connectNodes(fromID, toID string) bool {
    from, fromExists := ps.nodes[fromID]
    to, toExists := ps.nodes[toID]
    
    if !fromExists || !toExists {
        return false
    }
    
    // Adiciona conexão bidirecional
    from.Peers[toID] = &TestP2PPeer{
        ID:       toID,
        Address:  to.Address,
        Port:     to.Port,
        IsActive: true,
        LastSeen: time.Now(),
    }
    
    to.Peers[fromID] = &TestP2PPeer{
        ID:       fromID,
        Address:  from.Address,
        Port:     from.Port,
        IsActive: true,
        LastSeen: time.Now(),
    }
    
    return true
}

func (ps *mockP2PSystem) disconnectNode(nodeID string) int {
    node, exists := ps.nodes[nodeID]
    if !exists {
        return 0
    }
    
    // Remove conexão em todos os peers
    disconnectedCount := 0
    for peerID, _ := range node.Peers {
        if peer, exists := ps.nodes[peerID]; exists {
            delete(peer.Peers, nodeID)
            disconnectedCount++
        }
    }
    
    node.Peers = make(map[string]*TestP2PPeer)
    ps.disconnections += disconnectedCount
    return disconnectedCount
}

func (ps *mockP2PSystem) broadcastMessage(fromID string, msgType string, data interface{}) int {
    node, exists := ps.nodes[fromID]
    if !exists {
        return 0
    }
    
    sentCount := 0
    for peerID, peer := range node.Peers {
        if !peer.IsActive {
            continue
        }
        
        ps.messages = append(ps.messages, TestNetworkMessage{
            Type:      msgType,
            From:      fromID,
            To:        peerID,
            Data:      data,
            Timestamp: time.Now(),
        })
        sentCount++
    }
    
    return sentCount
}

func (ps *mockP2PSystem) synchronizeBlockchain(nodeID string) bool {
    node, exists := ps.nodes[nodeID]
    if !exists {
        return false
    }
    
    ps.syncAttempts++
    
    // Encontra peer com blockchain mais longa
    var bestPeer *TestP2PPeer
    maxLength := len(node.Blockchain)
    
    for peerID, peer := range node.Peers {
        if !peer.IsActive {
            continue
        }
        
        if peerNode, exists := ps.nodes[peerID]; exists {
            if len(peerNode.Blockchain) > maxLength {
                maxLength = len(peerNode.Blockchain)
                bestPeer = peer
            }
        }
    }
    
    // Se encontrou peer com blockchain maior, sincroniza
    if bestPeer != nil {
        bestNode := ps.nodes[bestPeer.ID]
        node.Blockchain = make([]*TestBlock, len(bestNode.Blockchain))
        copy(node.Blockchain, bestNode.Blockchain)
        ps.successfulSyncs++
        return true
    }
    
    return false
}

// Mock da descoberta bootstrap
type mockBootstrapDiscovery struct {
    hardcodedNodes []string
    dnsSeedNodes   []string
    scanResults    []string
}

func newMockBootstrapDiscovery() *mockBootstrapDiscovery {
    return &mockBootstrapDiscovery{
        hardcodedNodes: []string{"node1", "node2", "node3"},
        dnsSeedNodes:   []string{"seed1", "seed2"},
        scanResults:    []string{},
    }
}

func (bd *mockBootstrapDiscovery) findPeers() []string {
    // Simula descoberta baseada em DNS seeds
    return append(bd.hardcodedNodes, bd.dnsSeedNodes...)
}

func (bd *mockBootstrapDiscovery) scanLocalNetwork() []string {
    // Simula descoberta na rede local
    bd.scanResults = []string{"local1", "local2"}
    return bd.scanResults
}

// Testes para rede P2P
func TestNodeConnections(t *testing.T) {
    ps := newMockP2PSystem()
    
    // Cria nós
    node1 := ps.createNode("node1", "192.168.1.1", 8333)
    node2 := ps.createNode("node2", "192.168.1.2", 8333)
    node3 := ps.createNode("node3", "192.168.1.3", 8333)
    
    // Conecta nós
    success := ps.connectNodes("node1", "node2")
    if !success {
        t.Error("Conexão entre nós falhou")
    }
    
    if len(node1.Peers) != 1 || len(node2.Peers) != 1 {
        t.Error("Conexão bidirecional não foi estabelecida corretamente")
    }
    
    // Conecta mais nós
    ps.connectNodes("node1", "node3")
    ps.connectNodes("node2", "node3")
    
    if len(node1.Peers) != 2 {
        t.Error("node1 deveria ter 2 peers")
    }
    
    if len(node3.Peers) != 2 {
        t.Error("node3 deveria ter 2 peers")
    }
    
    // Teste de desconexão
    disconnected := ps.disconnectNode("node1")
    if disconnected != 2 {
        t.Errorf("Desconexão retornou count incorreto: %d", disconnected)
    }
    
    if len(node2.Peers) != 1 || node2.Peers["node3"] == nil {
        t.Error("Desconexão de node1 não manteve conexão entre node2 e node3")
    }
    
    if len(node1.Peers) != 0 {
        t.Error("node1 ainda tem peers após desconexão")
    }
}

func TestMessageBroadcasting(t *testing.T) {
    ps := newMockP2PSystem()
    
    // Configura rede em estrela (node1 no centro)
    ps.createNode("node1", "192.168.1.1", 8333)
    ps.createNode("node2", "192.168.1.2", 8333)
    ps.createNode("node3", "192.168.1.3", 8333)
    ps.createNode("node4", "192.168.1.4", 8333)
    
    ps.connectNodes("node1", "node2")
    ps.connectNodes("node1", "node3")
    ps.connectNodes("node1", "node4")
    
    // Broadcast de mensagem
    sentCount := ps.broadcastMessage("node1", "NEW_BLOCK", "blockhash123")
    
    if sentCount != 3 {
        t.Errorf("Broadcast deveria ter enviado para 3 peers, enviou para %d", sentCount)
    }
    
    if len(ps.messages) != 3 {
        t.Errorf("Número incorreto de mensagens registradas: %d", len(ps.messages))
    }
    
    // Testa se node4 recebeu mensagem
    receivedByNode4 := false
    for _, msg := range ps.messages {
        if msg.To == "node4" && msg.Type == "NEW_BLOCK" {
            receivedByNode4 = true
            break
        }
    }
    
    if !receivedByNode4 {
        t.Error("node4 não recebeu a mensagem de broadcast")
    }
}

func TestBlockchainSynchronization(t *testing.T) {
    ps := newMockP2PSystem()
    
    // Cria nós
    node1 := ps.createNode("node1", "192.168.1.1", 8333)
    node2 := ps.createNode("node2", "192.168.1.2", 8333)
    
    // Conecta nós
    ps.connectNodes("node1", "node2")
    
    // Adiciona blockchain mais longa ao node2
    node2.Blockchain = []*TestBlock{
        {Index: 1, Hash: "hash1"},
        {Index: 2, Hash: "hash2"},
        {Index: 3, Hash: "hash3"},
    }
    
    // node1 tem blockchain mais curta
    node1.Blockchain = []*TestBlock{
        {Index: 1, Hash: "hash1"},
    }
    
    // Sincroniza node1 com a rede
    success := ps.synchronizeBlockchain("node1")
    if !success {
        t.Error("Sincronização falhou")
    }
    
    if len(node1.Blockchain) != 3 {
        t.Errorf("node1 não sincronizou para tamanho correto: %d", len(node1.Blockchain))
    }
    
    if node1.Blockchain[2].Hash != "hash3" {
        t.Error("Blockchain sincronizada não tem o bloco final correto")
    }
    
    if ps.syncAttempts != 1 || ps.successfulSyncs != 1 {
        t.Error("Contadores de sincronização não foram atualizados corretamente")
    }
}

func TestBootstrapDiscovery(t *testing.T) {
    bd := newMockBootstrapDiscovery()
    
    // Teste de descoberta de peers
    peers := bd.findPeers()
    if len(peers) != 5 { // 3 hardcoded + 2 DNS seeds
        t.Errorf("Número incorreto de peers descobertos: %d", len(peers))
    }
    
    // Teste de scan de rede local
    localPeers := bd.scanLocalNetwork()
    if len(localPeers) != 2 {
        t.Errorf("Número incorreto de peers locais descobertos: %d", len(localPeers))
    }
}