package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Token struct {
	Index           int      `json:"index"`
	Nonce           int      `json:"nonce"`
	Hash            string   `json:"hash"`
	HashParts       []string `json:"hash_parts"`
	Timestamp       string   `json:"timestamp"`
	ContainsSyra    bool     `json:"contains_syra"`
	Validator       string   `json:"validator,omitempty"`
	PrevHash        string   `json:"prev_hash,omitempty"`
	WalletAddress   string   `json:"wallet_address,omitempty"`
	WalletSignature string   `json:"wallet_signature,omitempty"`
	MinerID         string   `json:"miner_id,omitempty"`
}

type Wallet struct {
	UserID           string   `json:"user_id"`
	UniqueToken      string   `json:"unique_token"`
	Signature        string   `json:"signature"`
	ValidationSeq    string   `json:"validation_sequence"`
	Address          string   `json:"address"`
	Balance          int      `json:"balance"`
	RegisteredBlocks []string `json:"registered_blocks"`
}

func loadWallet(userID string) (*Wallet, error) {
	// Corrigindo o caminho para a pasta PWtSY
	filename := filepath.Join("..", "PWtSY", fmt.Sprintf("wallet_%s.json", userID))
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

func saveWallet(wallet *Wallet) error {
	// Corrigindo o caminho para a pasta PWtSY
	filename := filepath.Join("..", "PWtSY", fmt.Sprintf("wallet_%s.json", wallet.UserID))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(wallet)
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Uso: go run validator.go <hash_do_bloco> <usuario_validador> <wallet_signature>")
		return
	}
	hash := os.Args[1]
	dono := os.Args[2]
	walletSig := os.Args[3]

	// Carrega a carteira do usuário
	wallet, err := loadWallet(dono)
	if err != nil {
		fmt.Printf("Erro ao carregar carteira do usuário %s: %v\n", dono, err)
		return
	}

	// Verifica se a assinatura confere
	if wallet.Signature != walletSig {
		fmt.Println("Assinatura da carteira inválida!")
		return
	}

	// Abre o arquivo tokens.json (corrigindo o caminho)
	file, err := os.OpenFile("../tokens.json", os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir tokens.json:", err)
		return
	}
	defer file.Close()

	var tokens []Token
	if err := json.NewDecoder(file).Decode(&tokens); err != nil {
		fmt.Println("Erro ao decodificar tokens.json:", err)
		return
	}

	// Procura o bloco pelo hash e adiciona as informações da carteira
	var blocoValidado *Token
	for i := range tokens {
		if tokens[i].Hash == hash {
			tokens[i].Validator = dono
			tokens[i].WalletAddress = wallet.Address
			tokens[i].WalletSignature = wallet.Signature
			tokens[i].MinerID = wallet.UserID
			blocoValidado = &tokens[i]
			break
		}
	}

	if blocoValidado == nil {
		fmt.Println("Bloco não encontrado.")
		return
	}

	// Verifica a integridade da cadeia de blocos
	for i := 1; i < len(tokens); i++ {
		if tokens[i].PrevHash != tokens[i-1].Hash {
			fmt.Printf("Integridade quebrada no bloco %d!\n", tokens[i].Index)
			return
		}
	}

	// Adiciona o bloco à carteira do usuário
	wallet.RegisteredBlocks = append(wallet.RegisteredBlocks, hash)
	wallet.Balance++
	err = saveWallet(wallet)
	if err != nil {
		fmt.Printf("Erro ao atualizar carteira: %v\n", err)
		return
	}

	// Salva o bloco validado em um novo arquivo
	out, err := os.Create("bloco_validado.json")
	if err != nil {
		fmt.Println("Erro ao criar bloco_validado.json:", err)
		return
	}
	defer out.Close()

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(blocoValidado); err != nil {
		fmt.Println("Erro ao salvar bloco_validado.json:", err)
		return
	}

	// Salva todos os tokens (com o validator atualizado) de volta no tokens.json
	file.Truncate(0)
	file.Seek(0, 0)
	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokens); err != nil {
		fmt.Println("Erro ao atualizar tokens.json:", err)
		return
	}

	fmt.Printf("Bloco validado com sucesso!\n")
	fmt.Printf("Usuário: %s\n", dono)
	fmt.Printf("Endereço da Carteira: %s\n", wallet.Address)
	fmt.Printf("Novo Saldo: %d SYRA\n", wallet.Balance)
	fmt.Println("Bloco adicionado à carteira e tokens.json atualizado!")
}
