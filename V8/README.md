# PTW Blockchain System 🚀

O **PTW Blockchain** é um sistema blockchain completo, modular e seguro, desenvolvido em Go, com todos os pilares de uma blockchain moderna, incluindo rede P2P real, mineração dinâmica, transações assinadas com RSA, contratos inteligentes SyraScript, consenso Proof-of-Stake, auditoria avançada, segurança multicamadas e monitoramento.

---

## 📋 Visão Geral

O PTW Blockchain implementa:

- **Transações assinadas** com RSA 2048-bit e prevenção de replay attacks
- **Pool de transações validadas** com verificação automática de assinaturas
- **Rede P2P distribuída** com descoberta automática (bootstrap, DNS, DHT, scan local)
- **Sincronização automática** da blockchain entre nós
- **Mineração automática e manual** com dificuldade dinâmica e monitoramento em tempo real
- **Carteiras digitais** com KYC, QR Code, saldo e histórico
- **Consenso Proof-of-Stake (PoS)** com seleção baseada em stake e reputação
- **Contratos inteligentes** em SyraScript com VM própria
- **Auditoria e logs avançados** com relatórios de segurança e alertas críticos
- **Validação robusta** de blocos, transações e contratos
- **Segurança multicamadas**: TLS, rate limiting, blacklist, análise de comportamento
- **Monitoramento e relatórios**: dificuldade, pool de transações, auditoria, status de validadores

---

## 🆕 Novidades e Destaques

- **Transações assinadas com RSA real**: Toda transação é assinada e validada criptograficamente.
- **Mineração dinâmica**: Dificuldade ajustada automaticamente, monitoramento em tempo real (`mining/difficulty.go`, `mining/difficulty_monitor.go`).
- **Auditoria e segurança**: Logs estruturados, relatórios (`audit/audit_system.go`), alertas críticos e análise de risco.
- **Contratos inteligentes SyraScript**: Linguagem própria, VM segura, integração com blockchain (`contracts/syrascript/`).
- **Rede P2P avançada**: Descoberta automática (bootstrap, DNS, DHT, scan local), sincronização inteligente, heartbeat, blacklist.
- **Consenso PoS distribuído**: Seleção de validadores por stake e reputação, rounds de consenso, distribuição de recompensas (`consensus/pos/pos_consensus.go`, `consensus/distributed_pos.go`).
- **Carteiras com KYC e QR Code**: Criação, verificação, exportação e histórico de blocos (`PWtSY/wallet.go`).
- **Pool de transações**: Pool validado com replay protection e regras de negócio (`network/transaction_handler.go`).

---

## 🏗️ Estrutura do Projeto

```
ptw/
├── main.go                    # Minerador manual (legado)
├── tokens.json                # Blockchain principal
├── go.mod / go.sum            # Dependências
│
├── miner/
│   ├── miner.go               # Minerador manual
│   ├── auto-miner/
│   │   └── auto_miner.go      # Minerador automático com dificuldade dinâmica
│   └── secure-miner/
│       └── secure_miner.go    # Mineração com validação de transações
│
├── mining/
│   ├── difficulty.go          # Gerenciador de dificuldade dinâmica
│   └── difficulty_monitor.go  # Monitor em tempo real da dificuldade
│
├── transaction/
│   └── transaction.go         # Transações assinadas, verificação RSA, prevenção de replay
│
├── PWtSY/
│   ├── wallet.go              # Carteiras digitais, KYC, QR Code
│   ├── wallet_*.json          # Carteiras dos usuários
│   ├── keypair_*.json         # Chaves RSA dos usuários
│
├── crypto/
│   └── keypair.go             # Geração e verificação de chaves RSA
│
├── network/
│   ├── p2p_node.go            # Nó P2P completo (TLS, peers, sync, discovery)
│   ├── addr_manager.go        # Gerenciamento de endereços de peers
│   ├── bootstrap.go           # Bootstrap e descoberta de peers
│   ├── dns_seed.go            # DNS Seeder (descoberta global)
│   ├── dht.go                 # Tabela hash distribuída (DHT)
│   ├── transaction_handler.go # Pool de transações validadas
│   └── transaction_types.go   # Tipos de transações
│
├── P2P_client/
│   └── p2p_client.go          # Cliente P2P interativo (CLI)
│
├── sync/
│   └── blockchain_sync.go     # Sincronização inteligente da blockchain
│
├── valid/
│   └── validator.go           # Validação de blocos, contratos e integridade
│
├── consensus/
│   ├── distributed_pos.go     # Consenso distribuído
│   └── pos/
│       └── pos_consensus.go   # Algoritmo Proof-of-Stake
│
├── contracts/
│   ├── contract_cli.go        # CLI para contratos inteligentes
│   ├── contracts.json         # Contratos cadastrados
│   ├── manager/
│   │   └── contract_manager.go # Gerenciador de contratos
│   └── syrascript/
│       ├── *.go               # Interpretador SyraScript (lexer, parser, VM, etc)
│       └── README.md          # Documentação da linguagem SyraScript
│
├── audit/
│   └── audit_system.go        # Auditoria e relatórios de segurança
│
├── security/
│   └── advanced_security.go   # Rate limiting, blacklist, análise de comportamento
```

---

## 🚀 Guia Rápido de Uso

### 1. Carteiras e Chaves

```bash
cd PWtSY
go run wallet.go create Alice
go run wallet.go kyc Alice
cd ../crypto
go run keypair.go generate Alice
```

### 2. Pool de Validadores PoS

```bash
cd consensus/pos
go run pos_consensus.go add_validator Alice 50 SYRA...
```

### 3. Iniciar Rede P2P

```bash
cd P2P_client
go run p2p_client.go Alice 8080 start
```

### 4. Mineração Automática

```bash
cd miner/auto-miner
go run auto_miner.go Alice <assinatura_da_wallet>
```

### 5. Validação de Blocos

```bash
cd valid
go run validator.go <hash_do_bloco> Alice <assinatura_da_wallet>
```

### 6. Contratos Inteligentes

```bash
cd contracts
go run contract_cli.go create "MeuContrato" Alice exemplo.syra
go run contract_cli.go list
go run contract_cli.go execute <id_do_contrato>
```

### 7. Auditoria e Segurança

```bash
cd audit
go run audit_system.go report
cat ../security_alerts.log
```

---

## 📜 SyraScript: Contratos Inteligentes

- Linguagem própria para contratos inteligentes, inspirada em Go/JavaScript.
- Tipagem dinâmica, controle de gás, funções integradas para blockchain.
- Veja [`contracts/syrascript/README.md`](contracts/syrascript/README.md) para sintaxe, exemplos e integração.

---

## 📊 Monitoramento e Auditoria

- **Relatórios de segurança:** `go run audit_system.go report`
- **Monitor de dificuldade:** `go run mining/difficulty.go monitor`
- **Status do pool PoS:** `go run consensus/pos/pos_consensus.go pool_status`
- **Histórico de blocos:** `go run PWtSY/wallet.go blocks <user_id>`

---

## 🛡️ Segurança

- **TLS obrigatório** para conexões P2P
- **Rate limiting** e **blacklist** automáticos
- **Logs de auditoria** e **alertas críticos**
- **KYC obrigatório** para minerar/validar/transferir
- **Validação de assinaturas** em todas as transações
- **Proteção contra replay attacks** (nonce)
- **Auditoria e análise de comportamento** de peers

---

## 💡 Funcionalidades Avançadas

- **Mineração dinâmica:** Ajuste automático de dificuldade, monitoramento em tempo real, recompensa variável.
- **Sincronização inteligente:** Blockchain sincronizada entre nós, escolha da melhor cadeia, propagação automática.
- **Pool de transações:** Validação de assinaturas, proteção contra replay, regras de negócio, limpeza automática.
- **Consenso distribuído:** Proof-of-Stake, seleção de validadores, rounds de consenso, reputação.
- **Contratos inteligentes:** Execução segura, triggers, integração com saldo, transferência e logs.
- **Auditoria:** Logs estruturados, relatórios, alertas críticos, métricas de segurança.

---

## 🔍 Detalhes Técnicos

### Sistema de Mineração
O PTW implementa um sistema de mineração híbrido que combina elementos tradicionais do Proof-of-Work para geração de novos blocos, com validação através de Proof-of-Stake. Os mineradores buscam hashes que contenham a string "Syra" enquanto também satisfazem critérios de dificuldade dinâmica que se ajustam para manter o tempo médio de bloco em aproximadamente 2 minutos.

### Algoritmo de Consenso PoS
O sistema PoS seleciona validadores com base em uma combinação de:
- **Stake** (quantidade de tokens em jogo)
- **Reputação** (histórico de validações corretas)
- **Disponibilidade** (tempo desde última validação)

Para fins de segurança, o sistema sempre seleciona entre os top candidatos usando o hash do bloco como seed, garantindo determinismo e previsibilidade no processo.

### Implementação da Rede P2P
A rede P2P utiliza:
- **TLS obrigatório** para segurança das comunicações
- **Descoberta estilo Bitcoin** com seeds DNS, bootstrap nodes e local discovery
- **DHT** (Tabela Hash Distribuída) para escalabilidade
- **Blacklisting automático** de peers maliciosos
- **Sincronização inteligente** com economia de banda e verificação por etapas

### Carteiras e Criptografia
- **RSA 2048-bit** para assinaturas digitais
- **SHA-256** para hashing de blocos e transações
- **QR Codes** para compartilhamento seguro de carteiras
- **KYC integrado** para compliance regulatório

---

## ⚙️ Requisitos do Sistema

- **Go 1.19+**
- **Mínimo 2GB RAM** para operação de nó completo
- **500MB espaço em disco** para blockchain inicial 
- Conexão à internet para sincronização e participação na rede
- Pacote `github.com/skip2/go-qrcode` para geração de QR codes

---

## 📝 Exemplo de Contrato SyraScript

```javascript
// Contrato de transferência programada
let owner = "Alice";
let recipient = "Bob";
let amount = 50;

function canExecute() {
    // Executa após o bloco 1000
    return blockHeight() > 1000;
}

function execute() {
    if (canExecute()) {
        // Transfere tokens
        transfer(owner, recipient, amount);
        log("Transferência programada executada!");
        return true;
    }
    return false;
}

// Chamada principal
execute();
```

---

## 🌐 Arquitetura de Rede

A arquitetura do PTW foi projetada para ser resistente a ataques e altamente disponível:

1. **Camada de Transporte**: TLS/TCP com verificação de integridade
2. **Camada de Rede**: Descoberta de peers com múltiplos mecanismos
3. **Camada de Consenso**: PoS distribuído com verificação em duas fases
4. **Camada de Aplicação**: Validação de todas as transações e blocos
5. **Camada de Persistência**: Armazenamento seguro com verificação de integridade

A sincronização entre nós utiliza um algoritmo de "melhor cadeia" que considera vários fatores:
- Comprimento da cadeia (70%)
- Confiabilidade do peer (20%)
- Latência da rede (10%)

---

## 👨‍💻 Créditos

Desenvolvido por Syra_ e colaboradores.  
Projeto educacional/demonstrativo de blockchain avançado.

---

**Status: 100% funcional, seguro e pronto para demonstração!**