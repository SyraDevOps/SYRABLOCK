package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"ptw/contracts/manager"
	"ptw/contracts/syrascript"
)

func cliMain() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	contractsFile := "contracts.json"
	cm, err := manager.NewContractManager(contractsFile)
	if err != nil {
		fmt.Printf("Erro ao inicializar gerenciador: %v\n", err)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "create":
		if len(os.Args) < 5 {
			fmt.Println("Uso: contract create <nome> <dono> <arquivo_fonte>")
			os.Exit(1)
		}
		name := os.Args[2]
		owner := os.Args[3]
		sourceFile := os.Args[4]

		source, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			fmt.Printf("Erro ao ler arquivo: %v\n", err)
			os.Exit(1)
		}

		contract, err := cm.CreateContract(name, owner, string(source))
		if err != nil {
			fmt.Printf("Erro ao criar contrato: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Contrato criado com sucesso!\n")
		fmt.Printf("   ID: %s\n", contract.ID)
		fmt.Printf("   Nome: %s\n", contract.Name)
		fmt.Printf("   Proprietário: %s\n", contract.Owner)

	case "list":
		contracts := cm.ListContracts()

		fmt.Println("=== CONTRATOS INTELIGENTES ===")
		if len(contracts) == 0 {
			fmt.Println("Nenhum contrato encontrado")
			return
		}

		for _, contract := range contracts {
			fmt.Printf("ID: %s\n", contract.ID)
			fmt.Printf("  Nome: %s\n", contract.Name)
			fmt.Printf("  Proprietário: %s\n", contract.Owner)
			fmt.Printf("  Status: %s\n", contract.Status)
			fmt.Printf("  Criado em: %s\n", contract.CreatedAt.Format("02/01/2006 15:04:05"))

			if !contract.LastExecuted.IsZero() {
				fmt.Printf("  Última execução: %s\n", contract.LastExecuted.Format("02/01/2006 15:04:05"))
			}

			fmt.Println()
		}

	case "view":
		if len(os.Args) < 3 {
			fmt.Println("Uso: contract view <id>")
			os.Exit(1)
		}

		contractID := os.Args[2]
		contract, exists := cm.GetContract(contractID)
		if !exists {
			fmt.Printf("❌ Contrato não encontrado: %s\n", contractID)
			os.Exit(1)
		}

		fmt.Printf("=== CONTRATO: %s ===\n", contract.Name)
		fmt.Printf("ID: %s\n", contract.ID)
		fmt.Printf("Proprietário: %s\n", contract.Owner)
		fmt.Printf("Status: %s\n", contract.Status)
		fmt.Printf("Criado em: %s\n", contract.CreatedAt.Format("02/01/2006 15:04:05"))

		if !contract.LastExecuted.IsZero() {
			fmt.Printf("Última execução: %s\n", contract.LastExecuted.Format("02/01/2006 15:04:05"))
		}

		fmt.Println("\nCódigo fonte:")
		fmt.Printf("```\n%s\n```\n", contract.Source)

		if len(contract.Triggers) > 0 {
			fmt.Println("\nGatilhos:")
			for i, trigger := range contract.Triggers {
				fmt.Printf("  %d. Tipo: %s, Ativo: %t\n", i+1, trigger.Type, trigger.Active)
			}
		}

	case "execute":
		if len(os.Args) < 3 {
			fmt.Println("Uso: contract execute <id>")
			os.Exit(1)
		}

		contractID := os.Args[2]

		// Cria contexto para execução
		context := &syrascript.Context{
			BlockHeight:    100, // Exemplo
			BlockTimestamp: time.Now(),
			ContractOwner:  "", // Será preenchido dinamicamente
		}

		result, err := cm.ExecuteContract(contractID, context)
		if err != nil {
			fmt.Printf("❌ Erro ao executar contrato: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Contrato executado com sucesso!\n")
		fmt.Printf("Resultado: %s\n", result.Inspect())

	case "activate":
		if len(os.Args) < 3 {
			fmt.Println("Uso: contract activate <id>")
			os.Exit(1)
		}

		contractID := os.Args[2]
		err := cm.ActivateContract(contractID)
		if err != nil {
			fmt.Printf("❌ Erro ao ativar contrato: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Contrato %s ativado com sucesso!\n", contractID)

	case "deactivate":
		if len(os.Args) < 3 {
			fmt.Println("Uso: contract deactivate <id>")
			os.Exit(1)
		}

		contractID := os.Args[2]
		err := cm.DeactivateContract(contractID)
		if err != nil {
			fmt.Printf("❌ Erro ao desativar contrato: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Contrato %s desativado com sucesso!\n", contractID)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Uso: contract <comando> [argumentos]")
	fmt.Println("Comandos:")
	fmt.Println("  create <nome> <dono> <arquivo_fonte> - Cria um novo contrato")
	fmt.Println("  list                                 - Lista todos os contratos")
	fmt.Println("  view <id>                            - Exibe detalhes de um contrato")
	fmt.Println("  execute <id>                         - Executa um contrato")
	fmt.Println("  activate <id>                        - Ativa um contrato")
	fmt.Println("  deactivate <id>                      - Desativa um contrato")
}

func main() {
	cliMain()
}
