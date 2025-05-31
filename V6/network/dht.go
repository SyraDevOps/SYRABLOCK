package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DHTNode representa um peer na DHT
type DHTNode struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"last_seen"`
}

// DHTTable é a tabela distribuída de peers
type DHTTable struct {
	mtx   sync.RWMutex
	nodes map[string]*DHTNode // key = nodeID
	self  *DHTNode
	file  string
}

// Novo DHTTable
func NewDHTTable(selfID, address string, port int, dataDir string) *DHTTable {
	self := &DHTNode{
		ID:       selfID,
		Address:  address,
		Port:     port,
		LastSeen: time.Now(),
	}
	dht := &DHTTable{
		nodes: make(map[string]*DHTNode),
		self:  self,
		file:  filepath.Join(dataDir, "dht.json"),
	}
	dht.nodes[selfID] = self
	dht.load()
	return dht
}

// Salva DHT em disco
func (dht *DHTTable) save() {
	dht.mtx.RLock()
	defer dht.mtx.RUnlock()
	nodes := make([]*DHTNode, 0, len(dht.nodes))
	for _, n := range dht.nodes {
		nodes = append(nodes, n)
	}
	file, err := os.Create(dht.file)
	if err == nil {
		defer file.Close()
		json.NewEncoder(file).Encode(nodes)
	}
}

// Carrega DHT do disco
func (dht *DHTTable) load() {
	file, err := os.Open(dht.file)
	if err != nil {
		return
	}
	defer file.Close()
	var nodes []*DHTNode
	if err := json.NewDecoder(file).Decode(&nodes); err == nil {
		for _, n := range nodes {
			dht.nodes[n.ID] = n
		}
	}
}

// Adiciona/atualiza peer na DHT
func (dht *DHTTable) AddOrUpdate(node *DHTNode) {
	dht.mtx.Lock()
	defer dht.mtx.Unlock()
	node.LastSeen = time.Now()
	dht.nodes[node.ID] = node
	dht.save()
}

// Remove peer inativo
func (dht *DHTTable) Cleanup() {
	dht.mtx.Lock()
	defer dht.mtx.Unlock()
	now := time.Now()
	for id, n := range dht.nodes {
		if id == dht.self.ID {
			continue
		}
		if now.Sub(n.LastSeen) > 24*time.Hour {
			delete(dht.nodes, id)
		}
	}
	dht.save()
}

// Busca peers próximos (Kademlia XOR distance)
func (dht *DHTTable) FindClosest(targetID string, max int) []*DHTNode {
	dht.mtx.RLock()
	defer dht.mtx.RUnlock()
	type distNode struct {
		dist uint64
		node *DHTNode
	}
	var list []distNode
	target := sha1.Sum([]byte(targetID))
	for _, n := range dht.nodes {
		idHash := sha1.Sum([]byte(n.ID))
		dist := xorDistance(target[:], idHash[:])
		list = append(list, distNode{dist, n})
	}
	// Ordena por distância
	sort.Slice(list, func(i, j int) bool { return list[i].dist < list[j].dist })
	result := make([]*DHTNode, 0, max)
	for i := 0; i < len(list) && i < max; i++ {
		result = append(result, list[i].node)
	}
	return result
}

func xorDistance(a, b []byte) uint64 {
	var d uint64
	for i := 0; i < len(a) && i < 8; i++ {
		d = (d << 8) | uint64(a[i]^b[i])
	}
	return d
}

// Broadcast presence to DHT peers
func (dht *DHTTable) Announce(node *P2PNode) {
	peers := dht.FindClosest(dht.self.ID, 8)
	for _, peer := range peers {
		if peer.ID == dht.self.ID {
			continue
		}
		go sendDHTAnnounce(peer, dht.self)
	}
}

func sendDHTAnnounce(peer *DHTNode, self *DHTNode) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(peer.Address, fmt.Sprintf("%d", peer.Port)), 2*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	msg := map[string]interface{}{
		"type": "dht_announce",
		"node": self,
	}
	json.NewEncoder(conn).Encode(msg)
}
