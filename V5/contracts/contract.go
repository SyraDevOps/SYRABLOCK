package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

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

func SaveContract(contract *Contract) error {
	file, err := os.OpenFile("contracts.json", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	var contracts []Contract
	json.NewDecoder(file).Decode(&contracts)
	contracts = append(contracts, *contract)
	file.Seek(0, 0)
	file.Truncate(0)
	return json.NewEncoder(file).Encode(contracts)
}

func loadContracts() ([]Contract, error) {
	file, err := os.Open("contracts.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var contracts []Contract
	if err := json.NewDecoder(file).Decode(&contracts); err != nil {
		return nil, err
	}
	return contracts, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run contract.go <comando> [parametros]")
		fmt.Println("Comandos:")
		fmt.Println("  create <owner> <trigger_block> <target> <amount>")
		fmt.Println("  list")
		return
	}
	switch os.Args[1] {
	case "create":
		if len(os.Args) < 6 {
			fmt.Println("Uso: create <owner> <trigger_block> <target> <amount>")
			return
		}
		owner := os.Args[2]
		triggerBlock := os.Args[3]
		target := os.Args[4]
		amount, _ := strconv.Atoi(os.Args[5])
		contract := &Contract{
			ID:           fmt.Sprintf("C-%d", time.Now().UnixNano()),
			Owner:        owner,
			TriggerBlock: triggerBlock,
			Action:       "transfer",
			Target:       target,
			Amount:       amount,
			Active:       true,
			CreatedAt:    time.Now(),
		}
		if err := SaveContract(contract); err != nil {
			fmt.Println("Erro ao salvar contrato:", err)
		} else {
			fmt.Println("Contrato criado:", contract.ID)
		}
	case "list":
		contracts, err := loadContracts()
		if err != nil {
			fmt.Println("Erro ao carregar contratos:", err)
			return
		}
		for _, c := range contracts {
			fmt.Printf("ID: %s | Owner: %s | Trigger: %s | Target: %s | Amount: %d | Ativo: %v\n",
				c.ID, c.Owner, c.TriggerBlock, c.Target, c.Amount, c.Active)
		}
	default:
		fmt.Println("Comando não reconhecido")
	}
}
