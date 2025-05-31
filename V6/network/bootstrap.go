package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

// Lista de nós bootstrap hardcoded
var HardcodedBootstrapNodes = []struct {
	IP   string
	Port string
}{
	{"104.131.144.82", "8333"},
	{"157.90.123.123", "8333"},
	{"45.76.203.127", "8333"},
	{"88.198.70.28", "8333"},
	{"78.47.3.220", "8333"},
	{"178.62.80.20", "8333"},
	{"163.172.161.52", "8333"},
	{"159.89.167.143", "8333"},
	// ... mais nós hardcoded
	{"127.0.0.1", "8333"}, // Local para desenvolvimento
}

// Lista de portas famosas para escanear na rede local
var CommonPorts = []string{"8333", "8332", "8334", "18333", "18332"}

// BootstrapManager gerencia o processo de bootstrap
type BootstrapManager struct {
	addrManager    *AddrManager
	dnsSeeder      *DNSSeeder
	discoveryNodes []string
	localNetworks  []string
	maxConnections int
	connectedPeers int
	mutex          sync.RWMutex
	connectTimeout time.Duration
	connected      bool
}

// NewBootstrapManager cria um novo gerenciador de bootstrap
func NewBootstrapManager(addrManager *AddrManager, dnsSeeder *DNSSeeder) *BootstrapManager {
	return &BootstrapManager{
		addrManager:    addrManager,
		dnsSeeder:      dnsSeeder,
		discoveryNodes: make([]string, 0),
		localNetworks:  []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"},
		maxConnections: 8,
		connectTimeout: 5 * time.Second,
	}
}

// InitialConnection tenta estabelecer conexões iniciais
func (bm *BootstrapManager) InitialConnection(connectionCallback func(ip, port string) bool) {
	fmt.Println("🚀 Iniciando processo de bootstrap da rede...")

	// Registra nós hardcoded no addr manager
	for _, node := range HardcodedBootstrapNodes {
		bm.addrManager.AddAddress(node.IP, node.Port, "hardcoded")
	}

	// Tenta DNS seeds se tiver poucos ou nenhum peer conhecido
	if bm.addrManager.GetAddresses(10, true, true) == nil {
		bm.dnsSeeder.SeedFromDNS(true)

		// Espera um pouco para as consultas DNS terminarem
		time.Sleep(500 * time.Millisecond)

		// Para desenvolvimento/teste, cria seeds locais simulados
		if os.Getenv("PTW_DEV_MODE") == "1" {
			bm.dnsSeeder.RegisterLocalSeeds()
		}
	}

	// Fase 1: Tenta conectar a nós com boa reputação
	connected := bm.connectToGoodPeers(connectionCallback)

	// Fase 2: Se não conectou, tenta qualquer nó conhecido
	if !connected {
		connected = bm.connectToAnyPeers(connectionCallback)
	}

	// Fase 3: Se ainda não conectou, tenta scan local
	if !connected {
		go bm.scanLocalNetwork()

		// Fase 4: Último recurso - tenta seed nodes diretamente
		connected = bm.connectToHardcodedNodes(connectionCallback)
	}

	bm.connected = connected

	if connected {
		fmt.Println("✅ Bootstrap concluído - conectado à rede PTW")
	} else {
		fmt.Println("❌ Falha no bootstrap - não foi possível conectar a nenhum peer")
	}
}

// Tenta conectar a peers com boa reputação
func (bm *BootstrapManager) connectToGoodPeers(connectionCallback func(ip, port string) bool) bool {
	goodAddrs := bm.addrManager.GetGoodAddresses(20)
	fmt.Printf("🔍 Encontrados %d peers com boa reputação\n", len(goodAddrs))

	var wg sync.WaitGroup
	successChan := make(chan bool, len(goodAddrs))
	resultChan := make(chan bool, 1)

	// Função que encerra após primeiro sucesso
	go func() {
		success := false
		for s := range successChan {
			if s {
				success = true
				break
			}
		}
		resultChan <- success
	}()

	// Tenta conectar a cada endereço em paralelo
	for _, addr := range goodAddrs {
		wg.Add(1)
		go func(ip, port string) {
			defer wg.Done()

			fmt.Printf("🔌 Tentando conectar a %s:%s (boa reputação)\n", ip, port)
			connected := connectionCallback(ip, port)

			bm.addrManager.Attempt(ip, port, connected)
			if connected {
				fmt.Printf("✅ Conexão estabelecida com %s:%s\n", ip, port)
				bm.mutex.Lock()
				bm.connectedPeers++
				bm.mutex.Unlock()
			}

			successChan <- connected
		}(addr.IP, addr.Port)

		// Limite de 5 tentativas paralelas
		if len(goodAddrs) > 5 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Fecha o canal após todas tentativas
	go func() {
		wg.Wait()
		close(successChan)
	}()

	// Aguarda resultado
	return <-resultChan
}

// Tenta conectar a qualquer peer conhecido
func (bm *BootstrapManager) connectToAnyPeers(connectionCallback func(ip, port string) bool) bool {
	anyAddrs := bm.addrManager.GetAddresses(30, true, true)
	fmt.Printf("🔍 Tentando conectar a %d peers conhecidos\n", len(anyAddrs))

	var wg sync.WaitGroup
	successChan := make(chan bool, len(anyAddrs))
	resultChan := make(chan bool, 1)

	// Função que encerra após primeiro sucesso
	go func() {
		success := false
		for s := range successChan {
			if s {
				success = true
				break
			}
		}
		resultChan <- success
	}()

	// Tenta conectar a cada endereço em paralelo
	for _, addr := range anyAddrs {
		wg.Add(1)
		go func(ip, port string) {
			defer wg.Done()

			fmt.Printf("🔌 Tentando conectar a %s:%s\n", ip, port)
			connected := connectionCallback(ip, port)

			bm.addrManager.Attempt(ip, port, connected)
			if connected {
				fmt.Printf("✅ Conexão estabelecida com %s:%s\n", ip, port)
				bm.mutex.Lock()
				bm.connectedPeers++
				bm.mutex.Unlock()
			}

			successChan <- connected
		}(addr.IP, addr.Port)

		// Limite de 5 tentativas paralelas
		if len(anyAddrs) > 5 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Fecha o canal após todas tentativas
	go func() {
		wg.Wait()
		close(successChan)
	}()

	// Aguarda resultado
	return <-resultChan
}

// Conecta aos nós hardcoded (último recurso)
func (bm *BootstrapManager) connectToHardcodedNodes(connectionCallback func(ip, port string) bool) bool {
	fmt.Println("🔄 Tentando nós hardcoded como último recurso...")

	// Embaralha a lista para não tentar sempre na mesma ordem
	nodes := make([]struct{ IP, Port string }, len(HardcodedBootstrapNodes))
	copy(nodes, HardcodedBootstrapNodes)
	rand.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	for _, node := range nodes {
		fmt.Printf("🔌 Tentando nó hardcoded %s:%s\n", node.IP, node.Port)
		if connectionCallback(node.IP, node.Port) {
			fmt.Printf("✅ Conexão estabelecida com %s:%s\n", node.IP, node.Port)
			bm.addrManager.Attempt(node.IP, node.Port, true)
			return true
		}

		bm.addrManager.Attempt(node.IP, node.Port, false)
	}

	return false
}

// Faz scan da rede local em busca de peers
func (bm *BootstrapManager) scanLocalNetwork() {
	fmt.Println("🔎 Escaneando rede local em busca de peers...")

	myIP, _ := getLocalIP()
	if myIP == "" {
		return
	}

	// Extrai prefixo de rede
	parts := net.ParseIP(myIP).To4()
	if parts == nil {
		return
	}

	// Escaneia os 10 IPs vizinhos em ambas direções
	baseIP := (uint32(parts[0]) << 24) | (uint32(parts[1]) << 16) |
		(uint32(parts[2]) << 8) | uint32(parts[3])

	var wg sync.WaitGroup

	// Tenta IPs na vizinhança
	for i := uint32(1); i <= 20; i++ {
		offset := i / 2
		if i%2 == 0 {
			offset = -offset
		}

		newIP := baseIP + offset
		if newIP == baseIP {
			continue // Pula o próprio IP
		}

		// Converte de volta para string
		ipStr := fmt.Sprintf("%d.%d.%d.%d",
			(newIP>>24)&0xFF, (newIP>>16)&0xFF, (newIP>>8)&0xFF, newIP&0xFF)

		// Testa em cada porta comum
		for _, port := range CommonPorts {
			wg.Add(1)
			go func(ip, port string) {
				defer wg.Done()

				conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), 500*time.Millisecond)
				if err == nil {
					conn.Close()
					fmt.Printf("🔍 Possível peer encontrado na rede local: %s:%s\n", ip, port)
					bm.addrManager.AddAddress(ip, port, "local")
				}
			}(ipStr, port)
		}
	}

	wg.Wait()
}

// Obtém IP local
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("IP não encontrado")
}
