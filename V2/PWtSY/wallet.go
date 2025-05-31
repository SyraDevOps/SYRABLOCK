package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
)

type Wallet struct {
	UserID           string    `json:"user_id"`
	UniqueToken      string    `json:"unique_token"`
	Signature        string    `json:"signature"`
	ValidationSeq    string    `json:"validation_sequence"`
	CreationDate     time.Time `json:"creation_date"`
	Address          string    `json:"address"`
	Balance          int       `json:"balance"`
	RegisteredBlocks []string  `json:"registered_blocks"`
}

type WalletExport struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
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

func CreateWallet(userID string) (*Wallet, error) {
	if userID == "" {
		return nil, fmt.Errorf("User ID cannot be empty")
	}

	uniqueToken := generateSecureRandom(32)
	validationSeq := generateSecureRandom(16)
	signature := generateUniqueSignature(userID, uniqueToken, validationSeq)
	address := generateAddress(signature)

	wallet := &Wallet{
		UserID:           userID,
		UniqueToken:      uniqueToken,
		Signature:        signature,
		ValidationSeq:    validationSeq,
		CreationDate:     time.Now(),
		Address:          address,
		Balance:          0,
		RegisteredBlocks: []string{},
	}

	return wallet, nil
}

func (w *Wallet) GenerateQRCode() error {
	exportData := WalletExport{
		Address:   w.Address,
		Signature: w.Signature,
		UserID:    w.UserID,
		CreatedAt: w.CreationDate.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(exportData)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("wallet_%s_qr.png", w.UserID)
	err = qrcode.WriteFile(string(jsonData), qrcode.Medium, 256, filename)
	if err != nil {
		return err
	}

	fmt.Printf("QR Code gerado: %s\n", filename)
	return nil
}

func (w *Wallet) SaveWallet() error {
	filename := fmt.Sprintf("wallet_%s.json", w.UserID)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(w)
}

func LoadWallet(userID string) (*Wallet, error) {
	filename := fmt.Sprintf("wallet_%s.json", userID)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wallet Wallet
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&wallet)
	return &wallet, err
}

func (w *Wallet) AddBlockToWallet(blockHash string) {
	w.RegisteredBlocks = append(w.RegisteredBlocks, blockHash)
	w.Balance++
	w.SaveWallet()
}

func (w *Wallet) GetUserBlocks() []string {
	return w.RegisteredBlocks
}

func (w *Wallet) DisplayWallet() {
	fmt.Printf("\n=== CARTEIRA SYRA ===\n")
	fmt.Printf("Usuário: %s\n", w.UserID)
	fmt.Printf("Endereço: %s\n", w.Address)
	fmt.Printf("Saldo: %d SYRA\n", w.Balance)
	fmt.Printf("Criada em: %s\n", w.CreationDate.Format("02/01/2006 15:04:05"))
	fmt.Printf("Blocos Registrados: %d\n", len(w.RegisteredBlocks))
	fmt.Printf("Assinatura: %s...\n", w.Signature[:32])
	fmt.Printf("====================\n\n")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run wallet.go <comando> [parametros]")
		fmt.Println("Comandos:")
		fmt.Println("  create <user_id>     - Cria nova carteira")
		fmt.Println("  load <user_id>       - Carrega carteira existente")
		fmt.Println("  blocks <user_id>     - Mostra blocos do usuário")
		return
	}

	command := os.Args[1]

	switch command {
	case "create":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o user_id")
			return
		}
		userID := os.Args[2]

		wallet, err := CreateWallet(userID)
		if err != nil {
			fmt.Printf("Erro ao criar carteira: %v\n", err)
			return
		}

		err = wallet.SaveWallet()
		if err != nil {
			fmt.Printf("Erro ao salvar carteira: %v\n", err)
			return
		}

		err = wallet.GenerateQRCode()
		if err != nil {
			fmt.Printf("Erro ao gerar QR Code: %v\n", err)
		}

		wallet.DisplayWallet()
		fmt.Printf("Carteira criada com sucesso!\n")

	case "load":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o user_id")
			return
		}
		userID := os.Args[2]

		wallet, err := LoadWallet(userID)
		if err != nil {
			fmt.Printf("Erro ao carregar carteira: %v\n", err)
			return
		}

		wallet.DisplayWallet()

	case "blocks":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o user_id")
			return
		}
		userID := os.Args[2]

		wallet, err := LoadWallet(userID)
		if err != nil {
			fmt.Printf("Erro ao carregar carteira: %v\n", err)
			return
		}

		blocks := wallet.GetUserBlocks()
		fmt.Printf("Blocos registrados para %s:\n", userID)
		for i, block := range blocks {
			fmt.Printf("%d. %s\n", i+1, block)
		}

	default:
		fmt.Println("Comando não reconhecido")
	}
}
