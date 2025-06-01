package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Dummy definition for P2PNode
type P2PNode struct {
	ID   string
	Addr string
}

// Dummy definition for NetworkMessage
type NetworkMessage struct {
	Type      int
	Payload   string
	Signature string
}

// Dummy constant for MSG_NEW_BLOCK
const MSG_NEW_BLOCK = 1

type SecurityManager struct {
	node           *P2PNode
	trustedPeers   map[string]bool
	blacklistedIPs map[string]time.Time
	rateLimiter    map[string][]time.Time
}

func NewSecurityManager(node *P2PNode) *SecurityManager {
	return &SecurityManager{
		node:           node,
		trustedPeers:   make(map[string]bool),
		blacklistedIPs: make(map[string]time.Time),
		rateLimiter:    make(map[string][]time.Time),
	}
}

// Assinatura digital de mensagens
func (sm *SecurityManager) SignMessage(message *NetworkMessage, privateKey *rsa.PrivateKey) error {
	// Serializa a mensagem sem a assinatura
	message.Signature = ""
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Cria hash da mensagem
	hash := sha256.Sum256(messageBytes)

	// Assina o hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return err
	}

	message.Signature = base64.StdEncoding.EncodeToString(signature)
	return nil
}

// Verificação de assinatura
func (sm *SecurityManager) VerifyMessage(message *NetworkMessage, publicKey *rsa.PublicKey) bool {
	originalSig := message.Signature
	message.Signature = ""

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return false
	}

	hash := sha256.Sum256(messageBytes)

	signature, err := base64.StdEncoding.DecodeString(originalSig)
	if err != nil {
		return false
	}

	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	message.Signature = originalSig // Restaura assinatura

	return err == nil
}

// Rate limiting
func (sm *SecurityManager) CheckRateLimit(peerID string, maxRequests int, timeWindow time.Duration) bool {
	now := time.Now()

	// Limpa requests antigos
	if requests, exists := sm.rateLimiter[peerID]; exists {
		validRequests := []time.Time{}
		for _, reqTime := range requests {
			if now.Sub(reqTime) < timeWindow {
				validRequests = append(validRequests, reqTime)
			}
		}
		sm.rateLimiter[peerID] = validRequests
	}

	// Verifica limite
	if len(sm.rateLimiter[peerID]) >= maxRequests {
		return false
	}

	// Adiciona nova request
	sm.rateLimiter[peerID] = append(sm.rateLimiter[peerID], now)
	return true
}

// Anti-spam e detecção de ataques
func (sm *SecurityManager) AnalyzePeerBehavior(peerID string, msg *NetworkMessage) bool {
	// Verifica se está na blacklist
	if blacklistTime, exists := sm.blacklistedIPs[peerID]; exists {
		if time.Since(blacklistTime) < 24*time.Hour {
			return false
		}
		delete(sm.blacklistedIPs, peerID)
	}

	// Rate limiting: máximo 100 mensagens por minuto
	if !sm.CheckRateLimit(peerID, 100, time.Minute) {
		fmt.Printf("⚠️ Rate limit excedido para peer %s\n", peerID)
		sm.blacklistedIPs[peerID] = time.Now()
		return false
	}

	// Verifica padrões suspeitos
	if sm.detectSuspiciousPatterns(peerID, msg) {
		fmt.Printf("🚨 Comportamento suspeito detectado: %s\n", peerID)
		sm.blacklistedIPs[peerID] = time.Now()
		return false
	}

	return true
}

func (sm *SecurityManager) detectSuspiciousPatterns(peerID string, msg *NetworkMessage) bool {
	// Exemplo: detecta flood de blocos
	if msg.Type == MSG_NEW_BLOCK {
		now := time.Now()
		requests := sm.rateLimiter[peerID]
		count := 0
		for _, t := range requests {
			if now.Sub(t) < 10*time.Second {
				count++
			}
		}
		if count > 5 {
			return true // Flood de blocos
		}
	}
	return false
}
