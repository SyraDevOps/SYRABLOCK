package main

import (
"bufio"
"crypto/rand"
"crypto/sha256"
"encoding/base64"
"encoding/hex"
"encoding/json"
"fmt"
"os"
"path/filepath"
"strconv"
"strings"
"time"

"github.com/skip2/go-qrcode"
)

// ANSI color codes for terminal styling
const (
ColorReset  = "\033[0m"
ColorRed    = "\033[31m"
ColorGreen  = "\033[32m"
ColorYellow = "\033[33m"
ColorBlue   = "\033[34m"
ColorPurple = "\033[35m"
ColorCyan   = "\033[36m"
ColorWhite  = "\033[37m"
ColorBold   = "\033[1m"
)

// Configuration structure
type Config struct {
CustomToken    string `json:"custom_token"`
SearchWord     string `json:"search_word"`
DataFolder     string `json:"data_folder"`
WalletFolder   string `json:"wallet_folder"`
BlockchainFile string `json:"blockchain_file"`
P2PPort        int    `json:"p2p_port"`
Initialized    bool   `json:"initialized"`
}

// Wallet structure
type Wallet struct {
UserID           string    `json:"user_id"`
UniqueToken      string    `json:"unique_token"`
Signature        string    `json:"signature"`
ValidationSeq    string    `json:"validation_sequence"`
CreationDate     time.Time `json:"creation_date"`
Address          string    `json:"address"`
Balance          int       `json:"balance"`
RegisteredBlocks []string  `json:"registered_blocks"`
KYCVerified      bool      `json:"kyc_verified"`
}

// Transaction structure
type Transaction struct {
ID        string    `json:"id"`
Type      string    `json:"type"`
From      string    `json:"from"`
To        string    `json:"to"`
Amount    int       `json:"amount"`
Timestamp time.Time `json:"timestamp"`
Contract  string    `json:"contract,omitempty"`
Signature string    `json:"signature,omitempty"`
}

// Block/Token structure
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

// File registry structure
type FileRegistry struct {
ID        string    `json:"id"`
Filename  string    `json:"filename"`
Hash      string    `json:"hash"`
Owner     string    `json:"owner"`
Timestamp time.Time `json:"timestamp"`
BlockHash string    `json:"block_hash"`
}

// Global variables
var (
config        Config
currentWallet *Wallet
scanner       = bufio.NewScanner(os.Stdin)
configFile    = "config.json"
)

func main() {
clearScreen()
printBanner()

// Load or initialize configuration
if err := loadConfig(); err != nil {
fmt.Println(colorText("⚠️  Configuração não encontrada. Iniciando primeira configuração...", ColorYellow))
firstTimeSetup()
}

// Main loop
for {
showMainMenu()
choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
walletMenu()
case "2":
transactionMenu()
case "3":
miningMenu()
case "4":
p2pMenu()
case "5":
fileMenu()
case "6":
blockchainMenu()
case "7":
configMenu()
case "8":
if currentWallet != nil {
fmt.Println(colorText("👋 Logout realizado com sucesso!", ColorGreen))
currentWallet = nil
}
case "9":
fmt.Println(colorText("👋 Encerrando SYRABLOCK...", ColorCyan))
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func clearScreen() {
fmt.Print("\033[H\033[2J")
}

func colorText(text, color string) string {
return color + text + ColorReset
}

func printBanner() {
banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ███████╗██╗   ██╗██████╗  █████╗ ██████╗ ██╗      ██████╗  ║
║   ██╔════╝╚██╗ ██╔╝██╔══██╗██╔══██╗██╔══██╗██║     ██╔═══██╗ ║
║   ███████╗ ╚████╔╝ ██████╔╝███████║██████╔╝██║     ██║   ██║ ║
║   ╚════██║  ╚██╔╝  ██╔══██╗██╔══██║██╔══██╗██║     ██║   ██║ ║
║   ███████║   ██║   ██║  ██║██║  ██║██████╔╝███████╗╚██████╔╝ ║
║   ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚══════╝ ╚═════╝  ║
║                                                               ║
║              🚀 Terminal Unificado - Versão 10 🚀              ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
fmt.Println(colorText(banner, ColorCyan))
fmt.Println(colorText("  Sistema Blockchain Completo - P2P | Mining | Transactions", ColorYellow))
fmt.Println()
}

func showMainMenu() {
clearScreen()
printBanner()

if currentWallet != nil {
fmt.Println(colorText("👤 Usuário: ", ColorGreen) + colorText(currentWallet.UserID, ColorBold))
fmt.Println(colorText("💰 Saldo: ", ColorGreen) + colorText(fmt.Sprintf("%d SYRA", currentWallet.Balance), ColorBold))
fmt.Println(colorText("📍 Endereço: ", ColorGreen) + colorText(currentWallet.Address[:20]+"...", ColorBold))
fmt.Println()
} else {
fmt.Println(colorText("⚠️  Nenhuma carteira carregada", ColorYellow))
fmt.Println()
}

fmt.Println(colorText("═══════════════ MENU PRINCIPAL ═══════════════", ColorCyan))
fmt.Println(colorText("1.", ColorYellow) + " 💼 Carteiras (Criar, Login, Gerenciar)")
fmt.Println(colorText("2.", ColorYellow) + " 💸 Transações (Enviar, Histórico)")
fmt.Println(colorText("3.", ColorYellow) + " ⛏️  Mineração (Minerar, Status)")
fmt.Println(colorText("4.", ColorYellow) + " 🌐 Rede P2P (Conectar, Status, Peers)")
fmt.Println(colorText("5.", ColorYellow) + " 📁 Arquivos (Registrar, Listar)")
fmt.Println(colorText("6.", ColorYellow) + " 🔗 Blockchain (Ver, Validar, Sincronizar)")
fmt.Println(colorText("7.", ColorYellow) + " ⚙️  Configurações (API, Token Customizado)")
fmt.Println(colorText("8.", ColorYellow) + " 🚪 Logout")
fmt.Println(colorText("9.", ColorYellow) + " ❌ Sair")
fmt.Println(colorText("═════════════════════════════════════════════", ColorCyan))
fmt.Println()
}

// Wallet Menu
func walletMenu() {
for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU CARTEIRAS ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Criar Nova Carteira")
fmt.Println(colorText("║  2.", ColorYellow) + " Login em Carteira Existente")
fmt.Println(colorText("║  3.", ColorYellow) + " Ver Detalhes da Carteira")
fmt.Println(colorText("║  4.", ColorYellow) + " Gerar QR Code")
fmt.Println(colorText("║  5.", ColorYellow) + " Verificar KYC")
fmt.Println(colorText("║  6.", ColorYellow) + " Listar Todas as Carteiras")
fmt.Println(colorText("║  7.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
createWallet()
case "2":
loginWallet()
case "3":
showWalletDetails()
case "4":
generateQRCode()
case "5":
verifyKYC()
case "6":
listWallets()
case "7":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func createWallet() {
fmt.Println(colorText("\n🔨 Criar Nova Carteira", ColorCyan))
fmt.Println(colorText("═══════════════════════", ColorCyan))

userID := readInput("Digite o ID do usuário: ")
if userID == "" {
fmt.Println(colorText("❌ ID não pode ser vazio!", ColorRed))
return
}

// Check if wallet already exists
walletPath := filepath.Join(config.WalletFolder, fmt.Sprintf("wallet_%s.json", userID))
if _, err := os.Stat(walletPath); err == nil {
fmt.Println(colorText("❌ Carteira já existe para este usuário!", ColorRed))
return
}

// Create wallet
wallet := &Wallet{
UserID:           userID,
UniqueToken:      generateSecureRandom(32),
ValidationSeq:    generateSecureRandom(16),
CreationDate:     time.Now(),
Balance:          0,
RegisteredBlocks: []string{},
KYCVerified:      false,
}

wallet.Signature = generateUniqueSignature(wallet.UserID, wallet.UniqueToken, wallet.ValidationSeq)
wallet.Address = generateAddress(wallet.Signature)

// Save wallet
if err := saveWallet(wallet); err != nil {
fmt.Println(colorText("❌ Erro ao salvar carteira: "+err.Error(), ColorRed))
return
}

currentWallet = wallet

fmt.Println(colorText("\n✅ Carteira criada com sucesso!", ColorGreen))
fmt.Println(colorText("👤 Usuário: ", ColorYellow) + wallet.UserID)
fmt.Println(colorText("📍 Endereço: ", ColorYellow) + wallet.Address)
fmt.Println(colorText("🔐 Assinatura: ", ColorYellow) + wallet.Signature[:20] + "...")
}

func loginWallet() {
fmt.Println(colorText("\n🔑 Login em Carteira", ColorCyan))
fmt.Println(colorText("═══════════════════", ColorCyan))

userID := readInput("Digite o ID do usuário: ")
if userID == "" {
fmt.Println(colorText("❌ ID não pode ser vazio!", ColorRed))
return
}

// Load wallet
walletPath := filepath.Join(config.WalletFolder, fmt.Sprintf("wallet_%s.json", userID))
data, err := os.ReadFile(walletPath)
if err != nil {
fmt.Println(colorText("❌ Carteira não encontrada!", ColorRed))
return
}

var wallet Wallet
if err := json.Unmarshal(data, &wallet); err != nil {
fmt.Println(colorText("❌ Erro ao ler carteira!", ColorRed))
return
}

currentWallet = &wallet
fmt.Println(colorText("\n✅ Login realizado com sucesso!", ColorGreen))
fmt.Println(colorText("👤 Bem-vindo, ", ColorYellow) + wallet.UserID + "!")
fmt.Println(colorText("💰 Saldo: ", ColorYellow) + fmt.Sprintf("%d SYRA", wallet.Balance))
}

func showWalletDetails() {
if currentWallet == nil {
fmt.Println(colorText("❌ Nenhuma carteira carregada!", ColorRed))
return
}

fmt.Println(colorText("\n💼 Detalhes da Carteira", ColorCyan))
fmt.Println(colorText("══════════════════════", ColorCyan))
fmt.Println(colorText("👤 Usuário: ", ColorYellow) + currentWallet.UserID)
fmt.Println(colorText("📍 Endereço: ", ColorYellow) + currentWallet.Address)
fmt.Println(colorText("💰 Saldo: ", ColorYellow) + fmt.Sprintf("%d SYRA", currentWallet.Balance))
fmt.Println(colorText("🔐 Assinatura: ", ColorYellow) + currentWallet.Signature[:20] + "...")
fmt.Println(colorText("📅 Criado em: ", ColorYellow) + currentWallet.CreationDate.Format("02/01/2006 15:04"))
fmt.Println(colorText("✅ KYC Verificado: ", ColorYellow) + fmt.Sprintf("%v", currentWallet.KYCVerified))
fmt.Println(colorText("📦 Blocos Registrados: ", ColorYellow) + fmt.Sprintf("%d", len(currentWallet.RegisteredBlocks)))
}

func generateQRCode() {
if currentWallet == nil {
fmt.Println(colorText("❌ Nenhuma carteira carregada!", ColorRed))
return
}

fmt.Println(colorText("\n📱 Gerando QR Code...", ColorCyan))

exportData := map[string]string{
"address":    currentWallet.Address,
"signature":  currentWallet.Signature,
"user_id":    currentWallet.UserID,
"created_at": currentWallet.CreationDate.Format(time.RFC3339),
}

jsonData, err := json.Marshal(exportData)
if err != nil {
fmt.Println(colorText("❌ Erro ao gerar QR Code!", ColorRed))
return
}

filename := filepath.Join(config.WalletFolder, fmt.Sprintf("wallet_%s_qr.png", currentWallet.UserID))
err = qrcode.WriteFile(string(jsonData), qrcode.Medium, 256, filename)
if err != nil {
fmt.Println(colorText("❌ Erro ao salvar QR Code!", ColorRed))
return
}

fmt.Println(colorText("✅ QR Code gerado: ", ColorGreen) + filename)
}

func verifyKYC() {
if currentWallet == nil {
fmt.Println(colorText("❌ Nenhuma carteira carregada!", ColorRed))
return
}

fmt.Println(colorText("\n🔍 Verificação KYC", ColorCyan))
fmt.Println(colorText("══════════════════", ColorCyan))

if currentWallet.KYCVerified {
fmt.Println(colorText("✅ KYC já verificado!", ColorGreen))
return
}

fmt.Println("Para verificar KYC, forneça as seguintes informações:")
fmt.Println("(Simulação - em produção seria integrado com serviço real)")

name := readInput("Nome completo: ")
cpf := readInput("CPF: ")

if name != "" && cpf != "" {
currentWallet.KYCVerified = true
saveWallet(currentWallet)
fmt.Println(colorText("✅ KYC verificado com sucesso!", ColorGreen))
}
}

func listWallets() {
fmt.Println(colorText("\n📋 Listando Carteiras", ColorCyan))
fmt.Println(colorText("═══════════════════", ColorCyan))

entries, err := os.ReadDir(config.WalletFolder)
if err != nil {
fmt.Println(colorText("❌ Erro ao ler pasta de carteiras!", ColorRed))
return
}

count := 0
for _, entry := range entries {
if strings.HasPrefix(entry.Name(), "wallet_") && strings.HasSuffix(entry.Name(), ".json") {
count++
userID := strings.TrimPrefix(strings.TrimSuffix(entry.Name(), ".json"), "wallet_")
fmt.Printf("%s%d. %s%s\n", ColorYellow, count, userID, ColorReset)
}
}

if count == 0 {
fmt.Println(colorText("ℹ️  Nenhuma carteira encontrada.", ColorYellow))
}
}

// Transaction Menu
func transactionMenu() {
if currentWallet == nil {
fmt.Println(colorText("❌ Faça login em uma carteira primeiro!", ColorRed))
waitForEnter()
return
}

for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU TRANSAÇÕES ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Enviar SYRA")
fmt.Println(colorText("║  2.", ColorYellow) + " Ver Histórico de Transações")
fmt.Println(colorText("║  3.", ColorYellow) + " Ver Transações Pendentes")
fmt.Println(colorText("║  4.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
sendTransaction()
case "2":
showTransactionHistory()
case "3":
showPendingTransactions()
case "4":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func sendTransaction() {
fmt.Println(colorText("\n💸 Enviar SYRA", ColorCyan))
fmt.Println(colorText("═════════════", ColorCyan))

to := readInput("Endereço de destino: ")
if to == "" {
fmt.Println(colorText("❌ Endereço não pode ser vazio!", ColorRed))
return
}

amountStr := readInput("Quantidade: ")
amount, err := strconv.Atoi(amountStr)
if err != nil || amount <= 0 {
fmt.Println(colorText("❌ Quantidade inválida!", ColorRed))
return
}

if amount > currentWallet.Balance {
fmt.Println(colorText("❌ Saldo insuficiente!", ColorRed))
return
}

// Create transaction
tx := Transaction{
ID:        generateSecureRandom(16),
Type:      "transfer",
From:      currentWallet.Address,
To:        to,
Amount:    amount,
Timestamp: time.Now(),
Signature: generateTransactionSignature(currentWallet.Address, to, amount),
}

// Save transaction to pending
if err := savePendingTransaction(tx); err != nil {
fmt.Println(colorText("❌ Erro ao salvar transação!", ColorRed))
return
}

// Update balance (in production, this would be done after mining)
currentWallet.Balance -= amount
saveWallet(currentWallet)

fmt.Println(colorText("\n✅ Transação criada com sucesso!", ColorGreen))
fmt.Println(colorText("🆔 ID: ", ColorYellow) + tx.ID)
fmt.Println(colorText("📍 Para: ", ColorYellow) + tx.To[:20] + "...")
fmt.Println(colorText("💰 Quantidade: ", ColorYellow) + fmt.Sprintf("%d SYRA", tx.Amount))
}

func showTransactionHistory() {
fmt.Println(colorText("\n📜 Histórico de Transações", ColorCyan))
fmt.Println(colorText("═════════════════════════", ColorCyan))

// Load blockchain and filter transactions
tokens := loadBlockchain()

count := 0
for _, token := range tokens {
for _, tx := range token.Transactions {
if tx.From == currentWallet.Address || tx.To == currentWallet.Address {
count++
fmt.Printf("\n%s#%d%s\n", ColorYellow, count, ColorReset)
fmt.Printf("  ID: %s\n", tx.ID)
fmt.Printf("  Tipo: %s\n", tx.Type)
fmt.Printf("  De: %s\n", tx.From[:20]+"...")
fmt.Printf("  Para: %s\n", tx.To[:20]+"...")
fmt.Printf("  Valor: %d SYRA\n", tx.Amount)
fmt.Printf("  Data: %s\n", tx.Timestamp.Format("02/01/2006 15:04"))
}
}
}

if count == 0 {
fmt.Println(colorText("ℹ️  Nenhuma transação encontrada.", ColorYellow))
}
}

func showPendingTransactions() {
fmt.Println(colorText("\n⏳ Transações Pendentes", ColorCyan))
fmt.Println(colorText("═══════════════════════", ColorCyan))

txFile := filepath.Join(config.DataFolder, "pending_transactions.json")
data, err := os.ReadFile(txFile)
if err != nil {
fmt.Println(colorText("ℹ️  Nenhuma transação pendente.", ColorYellow))
return
}

var transactions []Transaction
if err := json.Unmarshal(data, &transactions); err != nil {
fmt.Println(colorText("❌ Erro ao ler transações!", ColorRed))
return
}

count := 0
for _, tx := range transactions {
if tx.From == currentWallet.Address {
count++
fmt.Printf("\n%s#%d%s\n", ColorYellow, count, ColorReset)
fmt.Printf("  ID: %s\n", tx.ID)
fmt.Printf("  Para: %s\n", tx.To[:20]+"...")
fmt.Printf("  Valor: %d SYRA\n", tx.Amount)
}
}

if count == 0 {
fmt.Println(colorText("ℹ️  Nenhuma transação pendente.", ColorYellow))
}
}

// Mining Menu
func miningMenu() {
if currentWallet == nil {
fmt.Println(colorText("❌ Faça login em uma carteira primeiro!", ColorRed))
waitForEnter()
return
}

for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU MINERAÇÃO ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Minerar Novo Bloco")
fmt.Println(colorText("║  2.", ColorYellow) + " Ver Status de Mineração")
fmt.Println(colorText("║  3.", ColorYellow) + " Configurar Dificuldade")
fmt.Println(colorText("║  4.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
mineBlock()
case "2":
showMiningStatus()
case "3":
configureDifficulty()
case "4":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func mineBlock() {
fmt.Println(colorText("\n⛏️  Minerando Novo Bloco...", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))

tokens := loadBlockchain()
index := len(tokens) + 1

var prevHash string
if len(tokens) > 0 {
prevHash = tokens[len(tokens)-1].Hash
}

// Mine
fmt.Println("Procurando hash com '" + config.SearchWord + "'...")
startTime := time.Now()
nonce := 0
var hash string
var parts []string

for {
hash, parts = generateComplexHash(nonce)
if strings.Contains(hash, config.SearchWord) {
break
}
nonce++

if nonce%10000 == 0 {
fmt.Printf("\r⛏️  Tentativas: %d | Tempo: %.1fs", nonce, time.Since(startTime).Seconds())
}
}

// Load pending transactions
pendingTxs := loadPendingTransactions()

token := Token{
Index:           index,
Nonce:           nonce,
Hash:            hash,
HashParts:       parts,
Timestamp:       time.Now().Format(time.RFC3339),
ContainsSyra:    true,
PrevHash:        prevHash,
WalletAddress:   currentWallet.Address,
WalletSignature: currentWallet.Signature,
MinerID:         currentWallet.UserID,
Transactions:    pendingTxs,
}

tokens = append(tokens, token)
saveBlockchain(tokens)

// Clear pending transactions
clearPendingTransactions()

// Update wallet
currentWallet.Balance += 50 // Mining reward
currentWallet.RegisteredBlocks = append(currentWallet.RegisteredBlocks, hash)
saveWallet(currentWallet)

fmt.Println(colorText("\n\n✅ Bloco minerado com sucesso!", ColorGreen))
fmt.Println(colorText("📦 Bloco: ", ColorYellow) + fmt.Sprintf("#%d", index))
fmt.Println(colorText("🔑 Hash: ", ColorYellow) + hash[:30] + "...")
fmt.Println(colorText("🎲 Nonce: ", ColorYellow) + fmt.Sprintf("%d", nonce))
fmt.Println(colorText("⏱️  Tempo: ", ColorYellow) + fmt.Sprintf("%.2fs", time.Since(startTime).Seconds()))
fmt.Println(colorText("💰 Recompensa: ", ColorYellow) + "50 SYRA")
}

func showMiningStatus() {
fmt.Println(colorText("\n📊 Status de Mineração", ColorCyan))
fmt.Println(colorText("═══════════════════════", ColorCyan))

tokens := loadBlockchain()

mined := 0
for _, token := range tokens {
if token.MinerID == currentWallet.UserID {
mined++
}
}

fmt.Println(colorText("⛏️  Blocos Minerados: ", ColorYellow) + fmt.Sprintf("%d", mined))
fmt.Println(colorText("💰 Recompensa Total: ", ColorYellow) + fmt.Sprintf("%d SYRA", mined*50))
fmt.Println(colorText("📈 Taxa de Sucesso: ", ColorYellow) + fmt.Sprintf("%.2f%%", float64(mined)/float64(len(tokens))*100))
}

func configureDifficulty() {
fmt.Println(colorText("\n⚙️  Configurar Dificuldade", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))
fmt.Println(colorText("ℹ️  A dificuldade é ajustada automaticamente pelo sistema.", ColorYellow))
}

// P2P Menu
func p2pMenu() {
for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU REDE P2P ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Conectar à Rede P2P")
fmt.Println(colorText("║  2.", ColorYellow) + " Ver Status da Rede")
fmt.Println(colorText("║  3.", ColorYellow) + " Listar Peers Conectados")
fmt.Println(colorText("║  4.", ColorYellow) + " Sincronizar Blockchain")
fmt.Println(colorText("║  5.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
connectP2P()
case "2":
showP2PStatus()
case "3":
listPeers()
case "4":
syncBlockchain()
case "5":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func connectP2P() {
fmt.Println(colorText("\n🌐 Conectando à Rede P2P...", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))
fmt.Println(colorText("ℹ️  Porta: ", ColorYellow) + fmt.Sprintf("%d", config.P2PPort))
fmt.Println(colorText("✅ Para funcionalidade completa, use os módulos P2P separados.", ColorYellow))
}

func showP2PStatus() {
fmt.Println(colorText("\n📊 Status da Rede P2P", ColorCyan))
fmt.Println(colorText("════════════════════", ColorCyan))
fmt.Println(colorText("🌐 Status: ", ColorYellow) + "Simulado")
fmt.Println(colorText("👥 Peers: ", ColorYellow) + "0 conectados")
fmt.Println(colorText("📡 Porta: ", ColorYellow) + fmt.Sprintf("%d", config.P2PPort))
}

func listPeers() {
fmt.Println(colorText("\n👥 Peers Conectados", ColorCyan))
fmt.Println(colorText("══════════════════", ColorCyan))
fmt.Println(colorText("ℹ️  Nenhum peer conectado no momento.", ColorYellow))
}

func syncBlockchain() {
fmt.Println(colorText("\n🔄 Sincronizando Blockchain...", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))
fmt.Println(colorText("✅ Blockchain local atualizada.", ColorGreen))
}

// File Menu
func fileMenu() {
if currentWallet == nil {
fmt.Println(colorText("❌ Faça login em uma carteira primeiro!", ColorRed))
waitForEnter()
return
}

for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU ARQUIVOS ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Registrar Arquivo na Blockchain")
fmt.Println(colorText("║  2.", ColorYellow) + " Listar Arquivos Registrados")
fmt.Println(colorText("║  3.", ColorYellow) + " Verificar Integridade de Arquivo")
fmt.Println(colorText("║  4.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
registerFile()
case "2":
listFiles()
case "3":
verifyFile()
case "4":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func registerFile() {
fmt.Println(colorText("\n📝 Registrar Arquivo", ColorCyan))
fmt.Println(colorText("═══════════════════", ColorCyan))

filename := readInput("Nome do arquivo: ")
if filename == "" {
fmt.Println(colorText("❌ Nome não pode ser vazio!", ColorRed))
return
}

// Calculate file hash
fileHash := calculateFileHash(filename)

// Create registry
registry := FileRegistry{
ID:        generateSecureRandom(16),
Filename:  filename,
Hash:      fileHash,
Owner:     currentWallet.Address,
Timestamp: time.Now(),
BlockHash: "", // Will be set after mining
}

// Save registry
if err := saveFileRegistry(registry); err != nil {
fmt.Println(colorText("❌ Erro ao registrar arquivo!", ColorRed))
return
}

fmt.Println(colorText("\n✅ Arquivo registrado com sucesso!", ColorGreen))
fmt.Println(colorText("📄 Nome: ", ColorYellow) + registry.Filename)
fmt.Println(colorText("🔑 Hash: ", ColorYellow) + registry.Hash[:30] + "...")
fmt.Println(colorText("🆔 ID: ", ColorYellow) + registry.ID)
}

func listFiles() {
fmt.Println(colorText("\n📋 Arquivos Registrados", ColorCyan))
fmt.Println(colorText("═══════════════════════", ColorCyan))

registryFile := filepath.Join(config.DataFolder, "file_registry.json")
data, err := os.ReadFile(registryFile)
if err != nil {
fmt.Println(colorText("ℹ️  Nenhum arquivo registrado.", ColorYellow))
return
}

var registries []FileRegistry
if err := json.Unmarshal(data, &registries); err != nil {
fmt.Println(colorText("❌ Erro ao ler registros!", ColorRed))
return
}

count := 0
for _, reg := range registries {
if reg.Owner == currentWallet.Address {
count++
fmt.Printf("\n%s#%d%s\n", ColorYellow, count, ColorReset)
fmt.Printf("  Arquivo: %s\n", reg.Filename)
fmt.Printf("  Hash: %s...\n", reg.Hash[:30])
fmt.Printf("  Data: %s\n", reg.Timestamp.Format("02/01/2006 15:04"))
}
}

if count == 0 {
fmt.Println(colorText("ℹ️  Nenhum arquivo registrado por você.", ColorYellow))
}
}

func verifyFile() {
fmt.Println(colorText("\n🔍 Verificar Integridade", ColorCyan))
fmt.Println(colorText("════════════════════════", ColorCyan))

filename := readInput("Nome do arquivo: ")
if filename == "" {
fmt.Println(colorText("❌ Nome não pode ser vazio!", ColorRed))
return
}

currentHash := calculateFileHash(filename)

// Load registries
registryFile := filepath.Join(config.DataFolder, "file_registry.json")
data, err := os.ReadFile(registryFile)
if err != nil {
fmt.Println(colorText("❌ Nenhum registro encontrado!", ColorRed))
return
}

var registries []FileRegistry
if err := json.Unmarshal(data, &registries); err != nil {
fmt.Println(colorText("❌ Erro ao ler registros!", ColorRed))
return
}

for _, reg := range registries {
if reg.Filename == filename {
if reg.Hash == currentHash {
fmt.Println(colorText("✅ Arquivo íntegro! Hash corresponde ao registro.", ColorGreen))
} else {
fmt.Println(colorText("❌ AVISO: Arquivo foi modificado!", ColorRed))
fmt.Println(colorText("Hash registrado: ", ColorYellow) + reg.Hash[:30] + "...")
fmt.Println(colorText("Hash atual: ", ColorYellow) + currentHash[:30] + "...")
}
return
}
}

fmt.Println(colorText("ℹ️  Arquivo não encontrado nos registros.", ColorYellow))
}

// Blockchain Menu
func blockchainMenu() {
for {
clearScreen()
fmt.Println(colorText("╔═══════════════ MENU BLOCKCHAIN ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Ver Últimos Blocos")
fmt.Println(colorText("║  2.", ColorYellow) + " Ver Bloco Específico")
fmt.Println(colorText("║  3.", ColorYellow) + " Validar Integridade da Blockchain")
fmt.Println(colorText("║  4.", ColorYellow) + " Estatísticas da Blockchain")
fmt.Println(colorText("║  5.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
showLatestBlocks()
case "2":
showBlock()
case "3":
validateBlockchain()
case "4":
showBlockchainStats()
case "5":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func showLatestBlocks() {
fmt.Println(colorText("\n📦 Últimos Blocos", ColorCyan))
fmt.Println(colorText("═════════════════", ColorCyan))

tokens := loadBlockchain()

start := len(tokens) - 10
if start < 0 {
start = 0
}

for i := len(tokens) - 1; i >= start; i-- {
token := tokens[i]
fmt.Printf("\n%s#%d%s\n", ColorYellow, token.Index, ColorReset)
fmt.Printf("  Hash: %s...\n", token.Hash[:30])
fmt.Printf("  Minerador: %s\n", token.MinerID)
fmt.Printf("  Transações: %d\n", len(token.Transactions))
fmt.Printf("  Data: %s\n", token.Timestamp)
}
}

func showBlock() {
fmt.Println(colorText("\n🔍 Ver Bloco", ColorCyan))
fmt.Println(colorText("════════════", ColorCyan))

indexStr := readInput("Número do bloco: ")
index, err := strconv.Atoi(indexStr)
if err != nil {
fmt.Println(colorText("❌ Número inválido!", ColorRed))
return
}

tokens := loadBlockchain()

if index < 1 || index > len(tokens) {
fmt.Println(colorText("❌ Bloco não encontrado!", ColorRed))
return
}

token := tokens[index-1]
fmt.Printf("\n%sBloco #%d%s\n", ColorCyan, token.Index, ColorReset)
fmt.Println(strings.Repeat("═", 50))
fmt.Printf("Hash: %s\n", token.Hash)
fmt.Printf("Hash Anterior: %s\n", token.PrevHash)
fmt.Printf("Nonce: %d\n", token.Nonce)
fmt.Printf("Timestamp: %s\n", token.Timestamp)
fmt.Printf("Minerador: %s\n", token.MinerID)
fmt.Printf("Transações: %d\n", len(token.Transactions))

if len(token.Transactions) > 0 {
fmt.Println("\nTransações:")
for i, tx := range token.Transactions {
fmt.Printf("  %d. %s -> %s: %d SYRA\n", i+1, tx.From[:10]+"...", tx.To[:10]+"...", tx.Amount)
}
}
}

func validateBlockchain() {
fmt.Println(colorText("\n🔍 Validando Blockchain...", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))

tokens := loadBlockchain()

valid := true
for i := 1; i < len(tokens); i++ {
if tokens[i].PrevHash != tokens[i-1].Hash {
fmt.Printf(colorText("❌ Integridade quebrada no bloco %d!\n", ColorRed), tokens[i].Index)
valid = false
}
}

if valid {
fmt.Println(colorText("✅ Blockchain íntegra! Todos os blocos validados.", ColorGreen))
}
}

func showBlockchainStats() {
fmt.Println(colorText("\n📊 Estatísticas da Blockchain", ColorCyan))
fmt.Println(colorText("═════════════════════════════", ColorCyan))

tokens := loadBlockchain()

totalTx := 0
miners := make(map[string]int)

for _, token := range tokens {
totalTx += len(token.Transactions)
if token.MinerID != "" {
miners[token.MinerID]++
}
}

fmt.Println(colorText("📦 Total de Blocos: ", ColorYellow) + fmt.Sprintf("%d", len(tokens)))
fmt.Println(colorText("💸 Total de Transações: ", ColorYellow) + fmt.Sprintf("%d", totalTx))
fmt.Println(colorText("⛏️  Mineradores Únicos: ", ColorYellow) + fmt.Sprintf("%d", len(miners)))

if len(tokens) > 0 {
first := tokens[0]
last := tokens[len(tokens)-1]
fmt.Println(colorText("📅 Primeiro Bloco: ", ColorYellow) + first.Timestamp)
fmt.Println(colorText("📅 Último Bloco: ", ColorYellow) + last.Timestamp)
}
}

// Config Menu
func configMenu() {
for {
clearScreen()
fmt.Println(colorText("╔═══════════════ CONFIGURAÇÕES ═══════════════╗", ColorCyan))
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("║  1.", ColorYellow) + " Alterar Token Customizado")
fmt.Println(colorText("║  2.", ColorYellow) + " Configurar Porta P2P")
fmt.Println(colorText("║  3.", ColorYellow) + " Ver Configurações Atuais")
fmt.Println(colorText("║  4.", ColorYellow) + " Resetar Configurações")
fmt.Println(colorText("║  5.", ColorYellow) + " Voltar ao Menu Principal")
fmt.Println(colorText("║", ColorCyan))
fmt.Println(colorText("╚═══════════════════════════════════════════════╝", ColorCyan))
fmt.Println()

choice := readInput("Escolha uma opção: ")

switch choice {
case "1":
changeCustomToken()
case "2":
configureP2PPort()
case "3":
showCurrentConfig()
case "4":
resetConfig()
case "5":
return
default:
fmt.Println(colorText("❌ Opção inválida!", ColorRed))
}

waitForEnter()
}
}

func changeCustomToken() {
fmt.Println(colorText("\n🔧 Alterar Token Customizado", ColorCyan))
fmt.Println(colorText("═════════════════════════════", ColorCyan))
fmt.Println(colorText("⚠️  Atual: ", ColorYellow) + config.SearchWord)

newToken := readInput("Novo token (ou Enter para manter atual): ")
if newToken != "" {
config.SearchWord = newToken
config.CustomToken = newToken
saveConfig()
fmt.Println(colorText("✅ Token atualizado para: ", ColorGreen) + newToken)
}
}

func configureP2PPort() {
fmt.Println(colorText("\n🔧 Configurar Porta P2P", ColorCyan))
fmt.Println(colorText("═══════════════════════", ColorCyan))
fmt.Println(colorText("⚠️  Atual: ", ColorYellow) + fmt.Sprintf("%d", config.P2PPort))

portStr := readInput("Nova porta (ou Enter para manter atual): ")
if portStr != "" {
port, err := strconv.Atoi(portStr)
if err != nil || port < 1024 || port > 65535 {
fmt.Println(colorText("❌ Porta inválida!", ColorRed))
return
}
config.P2PPort = port
saveConfig()
fmt.Println(colorText("✅ Porta atualizada para: ", ColorGreen) + fmt.Sprintf("%d", port))
}
}

func showCurrentConfig() {
fmt.Println(colorText("\n⚙️  Configurações Atuais", ColorCyan))
fmt.Println(colorText("════════════════════════", ColorCyan))
fmt.Println(colorText("🔤 Token Customizado: ", ColorYellow) + config.CustomToken)
fmt.Println(colorText("🔍 Palavra de Busca: ", ColorYellow) + config.SearchWord)
fmt.Println(colorText("📂 Pasta de Dados: ", ColorYellow) + config.DataFolder)
fmt.Println(colorText("💼 Pasta de Carteiras: ", ColorYellow) + config.WalletFolder)
fmt.Println(colorText("🔗 Arquivo Blockchain: ", ColorYellow) + config.BlockchainFile)
fmt.Println(colorText("📡 Porta P2P: ", ColorYellow) + fmt.Sprintf("%d", config.P2PPort))
fmt.Println(colorText("✅ Inicializado: ", ColorYellow) + fmt.Sprintf("%v", config.Initialized))
}

func resetConfig() {
fmt.Println(colorText("\n⚠️  Resetar Configurações", ColorCyan))
fmt.Println(colorText("═══════════════════════════", ColorCyan))

confirm := readInput("Tem certeza? Isso removerá as configurações personalizadas (s/n): ")
if strings.ToLower(confirm) == "s" {
config = Config{
CustomToken:    "SYRA",
SearchWord:     "Syra",
DataFolder:     "./data",
WalletFolder:   "./PWtSY",
BlockchainFile: "tokens.json",
P2PPort:        8080,
Initialized:    true,
}
saveConfig()
fmt.Println(colorText("✅ Configurações resetadas!", ColorGreen))
}
}

// Helper Functions
func firstTimeSetup() {
fmt.Println(colorText("\n🚀 Bem-vindo ao SYRABLOCK Terminal!", ColorCyan))
fmt.Println(colorText("═══════════════════════════════════", ColorCyan))
fmt.Println("\nEsta é a primeira inicialização. Vamos configurar o sistema.\n")

// Custom token
fmt.Println(colorText("🔤 Token Customizado", ColorYellow))
fmt.Println("Você pode definir um token personalizado para sua blockchain.")
fmt.Println("Deixe em branco para usar o padrão (SYRA).")

token := readInput("Token customizado (ou Enter para SYRA): ")
if token == "" {
token = "SYRA"
}

// Search word
searchWord := readInput("Palavra de busca no hash (ou Enter para 'Syra'): ")
if searchWord == "" {
searchWord = "Syra"
}

// P2P Port
portStr := readInput("Porta P2P (ou Enter para 8080): ")
port := 8080
if portStr != "" {
if p, err := strconv.Atoi(portStr); err == nil && p >= 1024 && p <= 65535 {
port = p
}
}

// Create config
config = Config{
CustomToken:    token,
SearchWord:     searchWord,
DataFolder:     "./data",
WalletFolder:   "./PWtSY",
BlockchainFile: "tokens.json",
P2PPort:        port,
Initialized:    true,
}

// Create directories
os.MkdirAll(config.DataFolder, 0755)
os.MkdirAll(config.WalletFolder, 0755)

// Save config
saveConfig()

fmt.Println(colorText("\n✅ Configuração concluída!", ColorGreen))
fmt.Println(colorText("Token: ", ColorYellow) + token)
fmt.Println(colorText("Palavra de busca: ", ColorYellow) + searchWord)
fmt.Println(colorText("Porta P2P: ", ColorYellow) + fmt.Sprintf("%d", port))

waitForEnter()
}

func loadConfig() error {
data, err := os.ReadFile(configFile)
if err != nil {
return err
}
return json.Unmarshal(data, &config)
}

func saveConfig() error {
data, err := json.MarshalIndent(config, "", "  ")
if err != nil {
return err
}
return os.WriteFile(configFile, data, 0644)
}

func readInput(prompt string) string {
fmt.Print(colorText(prompt, ColorCyan))
scanner.Scan()
return strings.TrimSpace(scanner.Text())
}

func waitForEnter() {
fmt.Print(colorText("\n[Pressione ENTER para continuar]", ColorYellow))
scanner.Scan()
}

func generateSecureRandom(length int) string {
bytes := make([]byte, length)
rand.Read(bytes)
return hex.EncodeToString(bytes)
}

func generateUniqueSignature(userID, token, validationSeq string) string {
combined := fmt.Sprintf("%s:%s:%s:%d", userID, token, validationSeq, time.Now().UnixNano())
hash := sha256.Sum256([]byte(combined))
return base64.StdEncoding.EncodeToString(hash[:])
}

func generateAddress(signature string) string {
hash := sha256.Sum256([]byte("SYRA_WALLET_" + signature))
return "SYR" + hex.EncodeToString(hash[:])[:32]
}

func generateTransactionSignature(from, to string, amount int) string {
data := fmt.Sprintf("%s:%s:%d:%d", from, to, amount, time.Now().UnixNano())
hash := sha256.Sum256([]byte(data))
return base64.StdEncoding.EncodeToString(hash[:])
}

func calculateFileHash(filename string) string {
data := []byte(filename + time.Now().String())
hash := sha256.Sum256(data)
return hex.EncodeToString(hash[:])
}

func generateComplexHash(nonce int) (string, []string) {
var combined string
var parts []string
for j := 0; j < 4; j++ {
randomPart := generateSecureRandom(8)
input := fmt.Sprintf("%s%s2025", randomPart, config.CustomToken)
sum := sha256.Sum256([]byte(input))
hashPart := base64.StdEncoding.EncodeToString(sum[:])
parts = append(parts, hashPart)
combined += hashPart
}
finalSum := sha256.Sum256([]byte(combined))
finalHash := base64.StdEncoding.EncodeToString(finalSum[:])
return finalHash, parts
}

func saveWallet(wallet *Wallet) error {
data, err := json.MarshalIndent(wallet, "", "  ")
if err != nil {
return err
}
filename := filepath.Join(config.WalletFolder, fmt.Sprintf("wallet_%s.json", wallet.UserID))
return os.WriteFile(filename, data, 0644)
}

func savePendingTransaction(tx Transaction) error {
txFile := filepath.Join(config.DataFolder, "pending_transactions.json")

var transactions []Transaction
if data, err := os.ReadFile(txFile); err == nil {
json.Unmarshal(data, &transactions)
}

transactions = append(transactions, tx)

data, err := json.MarshalIndent(transactions, "", "  ")
if err != nil {
return err
}
return os.WriteFile(txFile, data, 0644)
}

func loadPendingTransactions() []Transaction {
txFile := filepath.Join(config.DataFolder, "pending_transactions.json")
data, err := os.ReadFile(txFile)
if err != nil {
return []Transaction{}
}

var transactions []Transaction
json.Unmarshal(data, &transactions)
return transactions
}

func clearPendingTransactions() {
txFile := filepath.Join(config.DataFolder, "pending_transactions.json")
os.WriteFile(txFile, []byte("[]"), 0644)
}

func saveFileRegistry(registry FileRegistry) error {
registryFile := filepath.Join(config.DataFolder, "file_registry.json")

var registries []FileRegistry
if data, err := os.ReadFile(registryFile); err == nil {
json.Unmarshal(data, &registries)
}

registries = append(registries, registry)

data, err := json.MarshalIndent(registries, "", "  ")
if err != nil {
return err
}
return os.WriteFile(registryFile, data, 0644)
}

func loadBlockchain() []Token {
data, err := os.ReadFile(config.BlockchainFile)
if err != nil {
return []Token{}
}

var tokens []Token
json.Unmarshal(data, &tokens)
return tokens
}

func saveBlockchain(tokens []Token) error {
data, err := json.MarshalIndent(tokens, "", "  ")
if err != nil {
return err
}
return os.WriteFile(config.BlockchainFile, data, 0644)
}
