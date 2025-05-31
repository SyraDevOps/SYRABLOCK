package main

import (
	"fmt"
	"time"
)

// Token represents a block in the blockchain.
type Token struct {
	Hash           string
	PrevHash       string
	ContainsSyra   bool
	Transactions   []Transaction
	MinerSignature string
}

// Transaction is a placeholder for transaction structure.
type Transaction struct {
	ID     string
	Amount int
}

// P2PNode represents a node in the network.
type P2PNode struct {
	ID          string
	Peers       map[string]*Peer
	IsValidator bool
	Stake       int
}

// Dummy implementations for required methods.
func (node *P2PNode) BroadcastToNetwork(msgType int, data interface{}) {
	for id, peer := range node.Peers {
		if peer.IsActive {
			go node.sendToPeer(peer, &NetworkMessage{
				Type:      msgType,
				From:      node.ID,
				To:        id,
				Data:      data,
				Timestamp: time.Now(),
			})
		}
	}
}

// sendToPeer is a dummy implementation that simulates sending a message to a peer.
func (node *P2PNode) sendToPeer(peer *Peer, msg *NetworkMessage) {
	// In a real implementation, this would send the message over the network.
	fmt.Printf("📡 Enviando mensagem para peer %s: tipo %d\n", peer.ID, msg.Type)
}

func (node *P2PNode) selectTopValidators(validators []string, max int) []string {
	// Implementation placeholder: just return the first max validators
	if len(validators) > max {
		return validators[:max]
	}
	return validators
}

func (node *P2PNode) verifyBlockHash(block *Token) bool {
	// Implementation placeholder
	return true
}

func (node *P2PNode) verifyPrevHash(block *Token) bool {
	// Implementation placeholder
	return true
}

func (node *P2PNode) verifyTransactions(txs []Transaction) bool {
	// Implementation placeholder
	return true
}

func (node *P2PNode) verifyMinerSignature(block *Token) bool {
	// Implementation placeholder
	return true
}

// type ConsensusRound is defined elsewhere, so remove this duplicate definition.

type Peer struct {
	ID       string
	Stake    int
	IsActive bool
}

type NetworkMessage struct {
	Type      int
	From      string
	To        string
	Data      interface{}
	Timestamp time.Time
}

const (
	MSG_CONSENSUS_REQUEST = 1
	MSG_CONSENSUS_VOTE    = 2
)

type ConsensusRound struct {
	RoundID       string          `json:"round_id"`
	BlockHash     string          `json:"block_hash"`
	ProposedBy    string          `json:"proposed_by"`
	Validators    []string        `json:"validators"`
	Votes         map[string]bool `json:"votes"`
	RequiredVotes int             `json:"required_votes"`
	Status        string          `json:"status"` // PENDING, APPROVED, REJECTED
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	ProposedBlock *Token          `json:"proposed_block"`
}

func (node *P2PNode) StartConsensusRound(block *Token) *ConsensusRound {
	validators := node.selectValidators()

	round := &ConsensusRound{
		RoundID:       fmt.Sprintf("ROUND_%d", time.Now().UnixNano()),
		BlockHash:     block.Hash,
		ProposedBy:    node.ID,
		Validators:    validators,
		Votes:         make(map[string]bool),
		RequiredVotes: len(validators)*2/3 + 1, // 67% + 1
		Status:        "PENDING",
		StartTime:     time.Now(),
		ProposedBlock: block,
	}

	fmt.Printf("🗳️ Iniciando consenso: %s\n", round.RoundID)
	fmt.Printf("   Bloco: %s\n", block.Hash[:16])
	fmt.Printf("   Validadores: %d\n", len(validators))
	fmt.Printf("   Votos necessários: %d\n", round.RequiredVotes)

	// Envia pedido de consenso para todos os validadores
	node.BroadcastToNetwork(MSG_CONSENSUS_REQUEST, round)

	// Agenda timeout para o consenso
	go node.consensusTimeout(round, 30*time.Second)

	return round
}

func (node *P2PNode) selectValidators() []string {
	validators := []string{}

	// Inclui nós com stake
	for id, peer := range node.Peers {
		if peer.Stake >= 10 && peer.IsActive { // Mínimo 10 SYRA
			validators = append(validators, id)
		}
	}

	// Inclui a si mesmo se for validador
	if node.IsValidator && node.Stake >= 10 {
		validators = append(validators, node.ID)
	}

	// Seleciona máximo 21 validadores (número ímpar para evitar empates)
	if len(validators) > 21 {
		// Algoritmo de seleção baseado em stake
		validators = node.selectTopValidators(validators, 21)
	}

	return validators
}

func (node *P2PNode) handleConsensusRequest(msg *NetworkMessage) *NetworkMessage {
	round := msg.Data.(*ConsensusRound)

	// Verifica se é um validador elegível
	isValidator := false
	for _, validatorID := range round.Validators {
		if validatorID == node.ID {
			isValidator = true
			break
		}
	}

	if !isValidator {
		return nil
	}

	// Valida o bloco proposto
	vote := node.validateProposedBlock(round.ProposedBlock)

	fmt.Printf("🗳️ Votando no consenso %s: %v\n", round.RoundID[:8], vote)

	// Envia voto de volta
	return &NetworkMessage{
		Type: MSG_CONSENSUS_VOTE,
		From: node.ID,
		To:   round.ProposedBy,
		Data: map[string]interface{}{
			"round_id": round.RoundID,
			"vote":     vote,
			"voter":    node.ID,
		},
		Timestamp: time.Now(),
	}
}

// Example usage to ensure handleConsensusRequest (and thus validateProposedBlock) is used
func main() {
	node := &P2PNode{
		ID:          "node1",
		Peers:       make(map[string]*Peer),
		IsValidator: true,
		Stake:       20,
	}
	block := &Token{
		Hash:           "somehashvalue1234567890",
		PrevHash:       "prevhashvalue0987654321",
		ContainsSyra:   true,
		Transactions:   []Transaction{},
		MinerSignature: "signature",
	}
	round := node.StartConsensusRound(block)
	msg := &NetworkMessage{
		Type:      MSG_CONSENSUS_REQUEST,
		From:      "node2",
		To:        "node1",
		Data:      round,
		Timestamp: time.Now(),
	}
	node.handleConsensusRequest(msg)
}

func (node *P2PNode) validateProposedBlock(block *Token) bool {
	// Validações do bloco

	// 1. Verifica hash
	if !node.verifyBlockHash(block) {
		fmt.Println("❌ Hash inválido")
		return false
	}

	// 2. Verifica se contém "Syra"
	if !block.ContainsSyra {
		fmt.Println("❌ Bloco não contém 'Syra'")
		return false
	}

	// 3. Verifica prev_hash
	if !node.verifyPrevHash(block) {
		fmt.Println("❌ PrevHash inválido")
		return false
	}

	// 4. Verifica transações (se houver)
	if !node.verifyTransactions(block.Transactions) {
		fmt.Println("❌ Transações inválidas")
		return false
	}

	// 5. Verifica assinatura do minerador
	if !node.verifyMinerSignature(block) {
		fmt.Println("❌ Assinatura do minerador inválida")
		return false
	}

	fmt.Println("✅ Bloco válido")
	return true
}

func (node *P2PNode) consensusTimeout(round *ConsensusRound, timeout time.Duration) {
	time.Sleep(timeout)

	if round.Status == "PENDING" {
		fmt.Printf("⏰ Timeout do consenso %s\n", round.RoundID[:8])
		round.Status = "TIMEOUT"
		round.EndTime = time.Now()
	}
}
