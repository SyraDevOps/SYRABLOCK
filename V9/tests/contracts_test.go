package tests

import (
	"testing"
	"time"
)

// Estruturas de teste para contratos inteligentes
type TestContract struct {
	ID           string
	Name         string
	Owner        string
	Source       string
	Status       string
	CreatedAt    time.Time
	LastExecuted time.Time
	Result       string
}

// Mock do sistema de contratos
type mockContractSystem struct {
	contracts   map[string]*TestContract
	execResults map[string]string // ID -> resultado
}

func newMockContractSystem() *mockContractSystem {
	return &mockContractSystem{
		contracts:   make(map[string]*TestContract),
		execResults: make(map[string]string),
	}
}

func (cs *mockContractSystem) createContract(name, owner, source string) *TestContract {
	contract := &TestContract{
		ID:        "contract_" + name + "_" + time.Now().Format("20060102150405"),
		Name:      name,
		Owner:     owner,
		Source:    source,
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
	}
	cs.contracts[contract.ID] = contract
	return contract
}

func (cs *mockContractSystem) executeContract(id string, args map[string]interface{}) (string, error) {
	contract, exists := cs.contracts[id]
	if !exists {
		return "", &mockError{"Contrato não encontrado"}
	}

	if contract.Status != "ACTIVE" {
		return "", &mockError{"Contrato inativo"}
	}

	// Simula execução analisando código fonte
	var result string

	// Simula execução baseada no código fonte
	if contractHasFunctionCall(contract.Source, "transfer") {
		if from, ok := args["from"].(string); ok {
			if to, ok := args["to"].(string); ok {
				result = "TRANSFER:" + from + ":" + to + ":10"
			}
		}
	} else if contractHasFunctionCall(contract.Source, "log") {
		result = "LOG:Contract executed successfully"
	} else {
		result = "true" // Execução padrão bem-sucedida
	}

	// Registra execução
	contract.LastExecuted = time.Now()
	cs.execResults[id] = result

	return result, nil
}

func (cs *mockContractSystem) deactivateContract(id string) error {
	contract, exists := cs.contracts[id]
	if !exists {
		return &mockError{"Contrato não encontrado"}
	}

	contract.Status = "INACTIVE"
	return nil
}

// Helper mock para verificar conteúdo do código fonte
func contractHasFunctionCall(source, functionName string) bool {
	// Simplificado: apenas verifica se a string contém o nome da função
	// Em um parser real, analisaria a AST
	return contains(source, functionName+"(")
}

// Helper para verificar substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Testes para contratos inteligentes
func TestContractCreation(t *testing.T) {
	cs := newMockContractSystem()

	source := `
        let balance = 100;
        
        function transfer(to, amount) {
            balance -= amount;
            return true;
        }
    `

	contract := cs.createContract("TestContract", "Alice", source)

	if contract.Name != "TestContract" || contract.Owner != "Alice" {
		t.Error("Criação de contrato falhou - dados básicos incorretos")
	}

	if contract.Status != "ACTIVE" {
		t.Error("Novo contrato deveria estar ativo")
	}

	if !contains(contract.Source, "function transfer") {
		t.Error("Código fonte do contrato não foi armazenado corretamente")
	}
}

func TestContractExecution(t *testing.T) {
	cs := newMockContractSystem()

	// Contrato de transferência
	transferContractSource := `
        function transfer(from, to, amount) {
            // Transfere tokens do remetente para o destinatário
            return true;
        }
    `

	transferContract := cs.createContract("TransferContract", "Alice", transferContractSource)

	// Executa contrato
	args := map[string]interface{}{
		"from": "Alice",
		"to":   "Bob",
	}

	result, err := cs.executeContract(transferContract.ID, args)
	if err != nil {
		t.Errorf("Execução de contrato falhou: %v", err)
	}

	if !contains(result, "TRANSFER:Alice:Bob") {
		t.Errorf("Resultado da execução não contém informação de transferência: %s", result)
	}

	// Verifica registro de execução
	if transferContract.LastExecuted.IsZero() {
		t.Error("Timestamp de última execução não foi registrado")
	}
}

func TestContractDeactivation(t *testing.T) {
	cs := newMockContractSystem()

	// Cria contrato
	contract := cs.createContract("DeactivationTest", "Alice", "function test() { return true; }")

	// Desativa contrato
	err := cs.deactivateContract(contract.ID)
	if err != nil {
		t.Errorf("Desativação de contrato falhou: %v", err)
	}

	if contract.Status != "INACTIVE" {
		t.Error("Contrato deveria estar inativo após desativação")
	}

	// Tenta executar contrato inativo
	_, err = cs.executeContract(contract.ID, nil)
	if err == nil {
		t.Error("Execução de contrato inativo deveria falhar")
	}

	// Tenta desativar contrato inexistente
	err = cs.deactivateContract("nonexistent_contract")
	if err == nil {
		t.Error("Desativação de contrato inexistente deveria falhar")
	}
}

func TestContractLogging(t *testing.T) {
	cs := newMockContractSystem()

	// Contrato com função de log
	logContractSource := `
        function execute() {
            log("Transaction completed");
            return true;
        }
    `

	logContract := cs.createContract("LogContract", "Alice", logContractSource)

	// Executa contrato
	result, err := cs.executeContract(logContract.ID, nil)
	if err != nil {
		t.Errorf("Execução de contrato falhou: %v", err)
	}

	if !contains(result, "LOG:") {
		t.Errorf("Resultado da execução não contém log: %s", result)
	}
}
