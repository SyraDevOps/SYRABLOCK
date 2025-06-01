package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Estrutura que representa um endereço de peer conhecido
type KnownAddress struct {
	IP              string    `json:"ip"`
	Port            string    `json:"port"`
	LastSeen        time.Time `json:"last_seen"`
	LastAttempt     time.Time `json:"last_attempt"`
	AttemptCount    int       `json:"attempt_count"`
	ConnectSuccess  int       `json:"connect_success"`
	ConnectFailures int       `json:"connect_failures"`
	Source          string    `json:"source"` // "dns", "peer", "local", "hardcoded"
	Services        uint64    `json:"services"`
	Banned          bool      `json:"banned"`
	BanExpires      time.Time `json:"ban_expires"`
}

// Gerenciador de endereços conhecido
type AddrManager struct {
	mtx             sync.RWMutex
	knownAddresses  map[string]*KnownAddress // key = "ip:port"
	filePath        string
	newAddresses    map[string]*KnownAddress // Endereços recentemente descobertos
	triedAddresses  map[string]*KnownAddress // Endereços já tentados
	maxNewBuckets   int
	maxTriedBuckets int
	saveInterval    time.Duration
	rand            *rand.Rand
}

// Cria novo gerenciador de endereços
func NewAddrManager(dataDir string) *AddrManager {
	// Cria diretório se não existir
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, 0700)
	}

	return &AddrManager{
		knownAddresses:  make(map[string]*KnownAddress),
		newAddresses:    make(map[string]*KnownAddress),
		triedAddresses:  make(map[string]*KnownAddress),
		filePath:        filepath.Join(dataDir, "peers.json"),
		maxNewBuckets:   1024,
		maxTriedBuckets: 256,
		saveInterval:    5 * time.Minute,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Inicia o gerenciador e carrega endereços do disco
func (am *AddrManager) Start() {
	// Carrega endereços salvos
	am.loadAddresses()

	// Inicia rotina de salvamento periódico
	go func() {
		ticker := time.NewTicker(am.saveInterval)
		defer ticker.Stop()

		for range ticker.C {
			am.saveAddresses()
		}
	}()
}

// Carrega endereços conhecidos do disco
func (am *AddrManager) loadAddresses() {
	am.mtx.Lock()
	defer am.mtx.Unlock()

	file, err := os.Open(am.filePath)
	if err != nil {
		fmt.Printf("📋 Arquivo de peers não encontrado, usando apenas bootstrap nodes\n")
		return
	}
	defer file.Close()

	var addresses []*KnownAddress
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&addresses); err != nil {
		fmt.Printf("❌ Erro ao decodificar arquivo de peers: %v\n", err)
		return
	}

	// Processa os endereços carregados
	for _, addr := range addresses {
		key := net.JoinHostPort(addr.IP, addr.Port)
		am.knownAddresses[key] = addr

		// Endereços que já foram conectados com sucesso vão para tried
		if addr.ConnectSuccess > 0 {
			am.triedAddresses[key] = addr
		} else {
			am.newAddresses[key] = addr
		}
	}

	fmt.Printf("📋 Carregados %d peers conhecidos do disco\n", len(addresses))
}

// Salva endereços conhecidos no disco
func (am *AddrManager) saveAddresses() {
	am.mtx.RLock()
	defer am.mtx.RUnlock()

	// Cria array com todos os endereços
	var addresses []*KnownAddress
	for _, addr := range am.knownAddresses {
		addresses = append(addresses, addr)
	}

	// Cria arquivo temporário primeiro
	tempFile := am.filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Printf("❌ Erro ao criar arquivo temporário: %v\n", err)
		return
	}

	// Serializa endereços
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(addresses)
	file.Close()

	if err != nil {
		fmt.Printf("❌ Erro ao serializar endereços: %v\n", err)
		os.Remove(tempFile)
		return
	}

	// Substitui arquivo original pelo temporário
	if err := os.Rename(tempFile, am.filePath); err != nil {
		fmt.Printf("❌ Erro ao substituir arquivo: %v\n", err)
		os.Remove(tempFile)
	}
}

// Adiciona um novo endereço
func (am *AddrManager) AddAddress(ip, port, source string) {
	am.mtx.Lock()
	defer am.mtx.Unlock()

	key := net.JoinHostPort(ip, port)

	// Verifica se já conhecemos este endereço
	if addr, exists := am.knownAddresses[key]; exists {
		// Atualiza fonte se for mais confiável
		if source == "hardcoded" || source == "dns" {
			addr.Source = source
		}
		return
	}

	// Cria novo endereço
	addr := &KnownAddress{
		IP:           ip,
		Port:         port,
		LastSeen:     time.Now(),
		Source:       source,
		AttemptCount: 0,
	}

	am.knownAddresses[key] = addr
	am.newAddresses[key] = addr
}

// Marca um endereço como tentado
func (am *AddrManager) Attempt(ip, port string, success bool) {
	am.mtx.Lock()
	defer am.mtx.Unlock()

	key := net.JoinHostPort(ip, port)
	addr, exists := am.knownAddresses[key]
	if !exists {
		return
	}

	addr.LastAttempt = time.Now()
	addr.AttemptCount++

	if success {
		addr.ConnectSuccess++
		addr.LastSeen = time.Now()

		// Mover para "tried" bucket
		delete(am.newAddresses, key)
		am.triedAddresses[key] = addr
	} else {
		addr.ConnectFailures++

		// Bane temporariamente se falhar muitas vezes consecutivas
		if addr.ConnectFailures > 5 && addr.ConnectFailures > addr.ConnectSuccess {
			addr.Banned = true
			addr.BanExpires = time.Now().Add(30 * time.Minute)
		}
	}
}

// Obtém um lote de endereços bons para tentar
func (am *AddrManager) GetAddresses(max int, includeNew, includeTried bool) []*KnownAddress {
	am.mtx.RLock()
	defer am.mtx.RUnlock()

	now := time.Now()
	result := make([]*KnownAddress, 0, max)
	added := make(map[string]bool)

	addFromMap := func(addresses map[string]*KnownAddress, count int) int {
		addedCount := 0
		for key, addr := range addresses {
			if len(result) >= max || addedCount >= count {
				break
			}

			// Pula endereços banidos
			if addr.Banned && now.Before(addr.BanExpires) {
				continue
			}

			// Evita duplicatas
			if added[key] {
				continue
			}

			result = append(result, addr)
			added[key] = true
			addedCount++
		}
		return addedCount
	}

	// Primeiro, adiciona endereços já testados com sucesso (mais confiáveis)
	if includeTried {
		addFromMap(am.triedAddresses, max/2)
	}

	// Depois, se precisar, adiciona novos endereços
	if includeNew && len(result) < max {
		remaining := max - len(result)
		addFromMap(am.newAddresses, remaining)
	}

	// Embaralha a lista para não favorecer sempre os mesmos peers
	am.rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})

	return result
}

// GetGoodAddresses retorna endereços com boa reputação
func (am *AddrManager) GetGoodAddresses(max int) []*KnownAddress {
	am.mtx.RLock()
	defer am.mtx.RUnlock()

	now := time.Now()
	result := make([]*KnownAddress, 0, max)
	added := make(map[string]bool)

	// Primeiro adiciona endereços "tried" com alta taxa de sucesso
	for key, addr := range am.triedAddresses {
		if addr.Banned && now.Before(addr.BanExpires) {
			continue
		}

		// Calcula pontuação de confiabilidade
		successRatio := float64(addr.ConnectSuccess) / float64(addr.AttemptCount+1)

		// Peers bons têm alto índice de sucesso
		if successRatio > 0.7 {
			result = append(result, addr)
			added[key] = true

			if len(result) >= max {
				break
			}
		}
	}

	// Se precisar de mais, adiciona de "new"
	if len(result) < max {
		// Prefere peers de fontes confiáveis
		for _, source := range []string{"hardcoded", "dns", "peer", "local"} {
			for key, addr := range am.newAddresses {
				if added[key] || len(result) >= max {
					continue
				}

				if addr.Source == source && (!addr.Banned || now.After(addr.BanExpires)) {
					result = append(result, addr)
					added[key] = true
				}
			}
		}
	}

	// Embaralha para evitar sempre os mesmos peers
	am.rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})

	return result
}

// Limpar endereços muito antigos ou com muitas falhas
func (am *AddrManager) CleanupAddresses() {
	am.mtx.Lock()
	defer am.mtx.Unlock()

	now := time.Now()
	removed := 0

	// Remove bans expirados
	for _, addr := range am.knownAddresses {
		if addr.Banned && now.After(addr.BanExpires) {
			addr.Banned = false
			fmt.Printf("🔓 Ban expirado para %s:%s\n", addr.IP, addr.Port)
		}
	}

	// Remove endereços muito velhos (6+ meses sem sucesso)
	staleThreshold := now.Add(-180 * 24 * time.Hour)

	for key, addr := range am.knownAddresses {
		// Remove se:
		// 1. Muito antigo sem tentativas
		// 2. Muitas falhas consecutivas
		// 3. Nunca teve sucesso e é muito antigo
		shouldRemove := false

		if addr.LastAttempt.Before(staleThreshold) && addr.ConnectSuccess == 0 {
			shouldRemove = true
		}

		if addr.ConnectFailures > 10 && addr.ConnectSuccess == 0 {
			shouldRemove = true
		}

		if shouldRemove {
			delete(am.knownAddresses, key)
			delete(am.newAddresses, key)
			delete(am.triedAddresses, key)
			removed++
		}
	}

	if removed > 0 {
		fmt.Printf("🧹 Limpeza concluída: %d endereços antigos removidos\n", removed)
	}
}
