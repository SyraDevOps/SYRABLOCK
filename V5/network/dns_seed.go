package main

import (
    "fmt"
    "net"
    "sync"
    "time"
)

// Lista de DNS seeds para PTW blockchain
var DNSSeeds = []string{
    // Em um sistema real, isso seriam domínios reais
    // Nesta implementação, usamos IPs diretamente como exemplo 
    "ptw-seed.example.com",
    "seed.ptw-network.org",
    "dns-seed.ptw-chain.net",
    "ptw-seeder.blockchain.info",
}

// Implementação do DNS Seeder - resolve DNS seeds para IPs
type DNSSeeder struct {
    seeds        []string
    addrManager  *AddrManager
    lastAttempt  time.Time
    seedInterval time.Duration
    mutex        sync.Mutex
}

// NewDNSSeeder cria um novo seeder
func NewDNSSeeder(addrManager *AddrManager) *DNSSeeder {
    return &DNSSeeder{
        seeds:        DNSSeeds,
        addrManager:  addrManager,
        seedInterval: 12 * time.Hour, // Tenta DNS seeds a cada 12 horas
    }
}

// TrySeedNodes tenta resolver os DNS seeds e adicionar ao addr manager
func (ds *DNSSeeder) TrySeedNodes() {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()
    
    // Evita consultas DNS muito frequentes
    if time.Since(ds.lastAttempt) < ds.seedInterval && ds.lastAttempt.Unix() > 0 {
        return
    }
    ds.lastAttempt = time.Now()
    
    fmt.Println("🌱 Consultando DNS seeds...")
    
    // Para cada DNS seed
    for _, seed := range ds.seeds {
        addrs, err := net.LookupHost(seed)
        if err != nil {
            fmt.Printf("❌ Não foi possível resolver DNS seed %s: %v\n", seed, err)
            continue
        }
        
        for _, addr := range addrs {
            // Verifica se é um IP válido
            if net.ParseIP(addr) != nil {
                // Adiciona com porta padrão
                ds.addrManager.AddAddress(addr, "8333", "dns")
            }
        }
        
        fmt.Printf("🌱 Obtidos %d IPs do DNS seed %s\n", len(addrs), seed)
    }
}

// Registrar IPs localmente para simular DNS seeds (para desenvolvimento)
func (ds *DNSSeeder) RegisterLocalSeeds() {
    // IPs que serão tratados como vindos dos DNS seeds
    hardcodedIPs := []string{
        "45.33.97.22", "134.209.101.19", "157.90.232.203",
        "168.119.99.128", "178.128.223.164", "193.135.10.186",
        "46.101.11.195", "5.9.121.164", "18.188.14.142",
        "104.248.139.211", "157.245.172.150", "167.99.174.238",
    }
    
    for _, ip := range hardcodedIPs {
        ds.addrManager.AddAddress(ip, "8333", "dns")
    }
    
    fmt.Printf("🌱 Registrados %d IPs simulando DNS seeds\n", len(hardcodedIPs))
}

// SeedFromDNS executa a descoberta de seeds DNS se necessário
func (ds *DNSSeeder) SeedFromDNS(force bool) {
    ds.mutex.Lock()
    shouldSeed := force || time.Since(ds.lastAttempt) >= ds.seedInterval
    ds.mutex.Unlock()
    
    if shouldSeed {
        go ds.TrySeedNodes()
    }
}