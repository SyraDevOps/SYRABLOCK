package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Transaction struct {
	Type      string    `json:"type"` // "transfer" ou "contract"
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int       `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract,omitempty"` // ID do contrato, se aplicável
}

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
	Transactions    []Transaction `json:"transactions,omitempty"` // NOVO
}

type Wallet struct {
	UserID           string   `json:"user_id"`
	UniqueToken      string   `json:"unique_token"`
	Signature        string   `json:"signature"`
	ValidationSeq    string   `json:"validation_sequence"`
	Address          string   `json:"address"`
	Balance          int      `json:"balance"`
	RegisteredBlocks []string `json:"registered_blocks"`
	KYCVerified      bool     `json:"kyc_verified"`
}

type Contract struct {
	ID           string    `json:"id"`
	Owner        string    `json:"owner"`
	TriggerBlock string    `json:"trigger_block"`
	Action       string    `json:"action"` // Exemplo: "transfer"
	Target       string    `json:"target"` // Usuário alvo
	Amount       int       `json:"amount"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
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

func loadContracts() ([]Contract, error) {
	file, err := os.Open("../contracts/contracts.json")
	if err != nil {
		return nil, nil // Não existe ainda, sem contratos
	}
	defer file.Close()
	var contracts []Contract
	json.NewDecoder(file).Decode(&contracts)
	return contracts, nil
}

func saveContracts(contracts []Contract) error {
	file, err := os.Create("../contracts/contracts.json")
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(contracts)
}

// Executa contratos automáticos ao validar bloco
func executeContracts(triggerBlock string, miner string, tokens []Token) {
	contracts, err := loadContracts()
	if err != nil {
		fmt.Println("Erro ao carregar contratos:", err)
		return
	}
	changed := false
	for i, c := range contracts {
		if c.Active && c.TriggerBlock == triggerBlock && c.Owner == miner && c.Action == "transfer" {
			fmt.Printf("Executando contrato %s: transferindo %d SYRA de %s para %s\n", c.ID, c.Amount, c.Owner, c.Target)
			err := Transfer(c.Owner, c.Target, c.Amount)
			if err != nil {
				fmt.Println("Erro ao executar contrato:", err)
			} else {
				contracts[i].Active = false // Desativa após execução
				changed = true
				// REGISTRA A TRANSAÇÃO NO BLOCO
				tx := Transaction{
					Type:      "contract",
					From:      c.Owner,
					To:        c.Target,
					Amount:    c.Amount,
					Timestamp: time.Now(),
					Contract:  c.ID,
				}
				for j := range tokens {
					if tokens[j].Hash == triggerBlock {
						tokens[j].Transactions = append(tokens[j].Transactions, tx)
						break
					}
				}
			}
		}
	}
	if changed {
		saveContracts(contracts)
	}
}

// Transferência entre carteiras (igual ao wallet.go)
func Transfer(fromID, toID string, amount int) error {
	from, err := loadWallet(fromID)
	if err != nil {
		return fmt.Errorf("remetente não encontrado")
	}
	to, err := loadWallet(toID)
	if err != nil {
		return fmt.Errorf("destinatário não encontrado")
	}
	if !from.KYCVerified || !to.KYCVerified {
		return fmt.Errorf("ambos usuários precisam de KYC")
	}
	if from.Balance < amount {
		return fmt.Errorf("saldo insuficiente")
	}
	from.Balance -= amount
	to.Balance += amount
	saveWallet(from)
	saveWallet(to)
	return nil
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

	// Verifica se o usuário passou no KYC
	if !wallet.KYCVerified {
		fmt.Println("Usuário não passou no KYC. Não pode validar blocos.")
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

	// Executa contratos automáticos, se houver
	executeContracts(hash, wallet.UserID, tokens)

	fmt.Printf("Bloco validado com sucesso!\n")
	fmt.Printf("Usuário: %s\n", dono)
	fmt.Printf("Endereço da Carteira: %s\n", wallet.Address)
	fmt.Printf("Novo Saldo: %d SYRA\n", wallet.Balance)
	fmt.Println("Bloco adicionado à carteira e tokens.json atualizado!")
}
