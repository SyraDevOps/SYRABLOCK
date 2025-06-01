package syrascript

import (
	"fmt"
	"strings"
	"time"
)

// Context contém o contexto de execução do contrato
type Context struct {
	BlockHeight    int
	BlockTimestamp time.Time
	ContractOwner  string
	TriggerEvent   string
	TriggerData    map[string]interface{}
}

// Contract representa um contrato compilado
type Contract struct {
	ID           string
	Name         string
	Owner        string
	Source       string
	CompiledAST  *Program
	CreatedAt    time.Time
	LastExecuted time.Time
	Status       string // "active", "inactive", "revoked"
	GasLimit     int
}

// Blockchain interface para interagir com o blockchain
type Blockchain interface {
	Transfer(from, to string, amount int) error
	GetBalance(userID string) (int, error)
	GetBlockHeight() int
	GetBlockTimestamp() time.Time
	Log(message string) error
}

// VM representa a máquina virtual para execução de contratos
type VM struct {
	blockchain Blockchain // Interface para acessar blockchain
	gasLimit   int        // Limite de gás (operações)
}

// NewVM cria uma nova VM
func NewVM(blockchain Blockchain, gasLimit int) *VM {
	return &VM{
		blockchain: blockchain,
		gasLimit:   gasLimit,
	}
}

// Compile compila código fonte em AST
func (vm *VM) Compile(source string) (*Program, error) {
	lexer := NewLexer(source)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		errorMsg := strings.Join(parser.Errors(), "\n")
		return nil, fmt.Errorf("erro de compilação:\n%s", errorMsg)
	}

	return program, nil
}

// ExecuteContract executa um contrato SyraScript
func (vm *VM) ExecuteContract(contract *Contract, context *Context) (Object, error) {
	// Criar avaliador com limite de gás
	evaluator := NewEvaluator(contract.GasLimit)

	// Executa o programa
	result := evaluator.Evaluate(contract.CompiledAST)

	// Processa o resultado para executar ações na blockchain
	if err := vm.processResult(result); err != nil {
		return nil, fmt.Errorf("erro ao processar resultado: %v", err)
	}

	return result, nil
}

// ProcessResult processa o resultado da execução do contrato
func (vm *VM) processResult(result Object) error {
	// Se for um erro, retorna-o
	if err, ok := result.(*Error); ok {
		return fmt.Errorf("erro de execução: %s", err.Message)
	}

	// Processa instruções especiais retornadas pelo contrato
	if str, ok := result.(*String); ok {
		if strings.HasPrefix(str.Value, "TRANSFER:") {
			parts := strings.Split(str.Value, ":")
			if len(parts) != 4 {
				return fmt.Errorf("formato de instrução TRANSFER inválido: %s", str.Value)
			}

			from, to := parts[1], parts[2]
			var amount int
			fmt.Sscanf(parts[3], "%d", &amount)

			return vm.blockchain.Transfer(from, to, amount)
		}

		if strings.HasPrefix(str.Value, "LOG:") {
			message := strings.TrimPrefix(str.Value, "LOG:")
			return vm.blockchain.Log(message)
		}
	}

	return nil
}
