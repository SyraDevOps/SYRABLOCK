package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ptw/contracts/syrascript"
)

// ContractManager gerencia contratos inteligentes
type ContractManager struct {
	contractsFile string
	contracts     map[string]*Contract
	vm            *syrascript.VM
	blockchain    *BlockchainAdapter
}

// Contract representa um contrato inteligente
type Contract struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Owner        string                 `json:"owner"`
	Source       string                 `json:"source"`
	CompiledAST  *syrascript.Program    `json:"-"` // Não serializado
	CreatedAt    time.Time              `json:"created_at"`
	LastExecuted time.Time              `json:"last_executed"`
	Status       string                 `json:"status"`
	GasLimit     int                    `json:"gas_limit"`
	Triggers     []Trigger              `json:"triggers"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Trigger define quando um contrato deve ser executado
type Trigger struct {
	Type      string                 `json:"type"`      // "block", "time", "event", "manual"
	Condition map[string]interface{} `json:"condition"` // Condições específicas do trigger
	Active    bool                   `json:"active"`
}

// BlockchainAdapter implementa a interface syrascript.Blockchain
type BlockchainAdapter struct {
	// Implementação real de interação com a blockchain
}

// Transfer implementa transferência de tokens
func (b *BlockchainAdapter) Transfer(from, to string, amount int) error {
	fmt.Printf("🔄 Transferência: %d SYRA de %s para %s\n", amount, from, to)
	// Aqui você integraria com o sistema real de blockchain
	return nil
}

// GetBalance implementa consulta de saldo
func (b *BlockchainAdapter) GetBalance(userID string) (int, error) {
	fmt.Printf("📊 Consultando saldo de %s\n", userID)
	// Aqui você integraria com o sistema real de blockchain
	return 1000, nil // Valor simulado
}

// GetBlockHeight implementa consulta de altura do bloco
func (b *BlockchainAdapter) GetBlockHeight() int {
	// Aqui você integraria com o sistema real de blockchain
	return 1500 // Valor simulado
}

// GetBlockTimestamp implementa consulta de timestamp do bloco
func (b *BlockchainAdapter) GetBlockTimestamp() time.Time {
	return time.Now()
}

// Log implementa logging na blockchain
func (b *BlockchainAdapter) Log(message string) error {
	fmt.Printf("📝 [Contract Log] %s\n", message)
	return nil
}

// NewContractManager cria um novo gerenciador de contratos
func NewContractManager(contractsFile string) (*ContractManager, error) {
	cm := &ContractManager{
		contractsFile: contractsFile,
		contracts:     make(map[string]*Contract),
	}

	// Inicializa o adaptador blockchain
	cm.blockchain = &BlockchainAdapter{}

	// Inicializa a VM
	cm.vm = syrascript.NewVM(cm.blockchain, 1000)

	// Carrega contratos existentes
	if err := cm.loadContracts(); err != nil {
		return nil, err
	}

	return cm, nil
}

// CreateContract cria um novo contrato
func (cm *ContractManager) CreateContract(name, owner, source string) (*Contract, error) {
	// Compila o código-fonte
	program, err := cm.vm.Compile(source)
	if err != nil {
		return nil, fmt.Errorf("erro de compilação: %v", err)
	}

	// Cria ID único para o contrato
	id := fmt.Sprintf("contract_%d", time.Now().UnixNano())

	contract := &Contract{
		ID:          id,
		Name:        name,
		Owner:       owner,
		Source:      source,
		CompiledAST: program,
		CreatedAt:   time.Now(),
		Status:      "active",
		GasLimit:    1000,
		Triggers:    []Trigger{},
		Metadata:    map[string]interface{}{},
	}

	// Adiciona à lista de contratos
	cm.contracts[id] = contract

	// Salva alterações
	if err := cm.saveContracts(); err != nil {
		return nil, err
	}

	return contract, nil
}

// ExecuteContract executa um contrato específico
func (cm *ContractManager) ExecuteContract(id string, context *syrascript.Context) (syrascript.Object, error) {
	contract, exists := cm.contracts[id]
	if !exists {
		return nil, fmt.Errorf("contrato não encontrado: %s", id)
	}

	if contract.Status != "active" {
		return nil, fmt.Errorf("contrato inativo: %s (status: %s)", id, contract.Status)
	}

	// Converte para o formato interno da VM
	vmContract := &syrascript.Contract{
		ID:           contract.ID,
		Name:         contract.Name,
		Owner:        contract.Owner,
		Source:       contract.Source,
		CompiledAST:  contract.CompiledAST,
		CreatedAt:    contract.CreatedAt,
		LastExecuted: contract.LastExecuted,
		Status:       contract.Status,
		GasLimit:     contract.GasLimit,
	}

	// Executa o contrato
	result, err := cm.vm.ExecuteContract(vmContract, context)
	if err != nil {
		return nil, err
	}

	// Atualiza última execução
	contract.LastExecuted = time.Now()
	cm.saveContracts()

	return result, nil
}

// loadContracts carrega contratos do arquivo
func (cm *ContractManager) loadContracts() error {
	// Verifica se o arquivo existe
	if _, err := os.Stat(cm.contractsFile); os.IsNotExist(err) {
		return nil // Arquivo não existe, começa com mapa vazio
	}

	// Abre o arquivo
	file, err := os.Open(cm.contractsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decodifica JSON
	var contracts []*Contract
	if err := json.NewDecoder(file).Decode(&contracts); err != nil {
		return err
	}

	// Compila a AST para cada contrato
	for _, contract := range contracts {
		program, err := cm.vm.Compile(contract.Source)
		if err != nil {
			return fmt.Errorf("erro ao recompilar contrato %s: %v", contract.ID, err)
		}
		contract.CompiledAST = program
		cm.contracts[contract.ID] = contract
	}

	return nil
}

// saveContracts salva contratos no arquivo
func (cm *ContractManager) saveContracts() error {
	// Cria diretório se necessário
	dir := filepath.Dir(cm.contractsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Abre arquivo para escrita
	file, err := os.Create(cm.contractsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Prepara lista de contratos para serialização
	var contractsList []*Contract
	for _, c := range cm.contracts {
		contractsList = append(contractsList, c)
	}

	// Codifica e salva em JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(contractsList)
}

// GetContract retorna um contrato pelo ID
func (cm *ContractManager) GetContract(id string) (*Contract, bool) {
	contract, exists := cm.contracts[id]
	return contract, exists
}

// ListContracts retorna todos os contratos
func (cm *ContractManager) ListContracts() []*Contract {
	var contracts []*Contract
	for _, contract := range cm.contracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// ActivateContract ativa um contrato
func (cm *ContractManager) ActivateContract(id string) error {
	contract, exists := cm.contracts[id]
	if !exists {
		return fmt.Errorf("contrato não encontrado: %s", id)
	}

	contract.Status = "active"
	return cm.saveContracts()
}

// DeactivateContract desativa um contrato
func (cm *ContractManager) DeactivateContract(id string) error {
	contract, exists := cm.contracts[id]
	if !exists {
		return fmt.Errorf("contrato não encontrado: %s", id)
	}

	contract.Status = "inactive"
	return cm.saveContracts()
}

// RevokeContract revoga um contrato (não pode ser reativado)
func (cm *ContractManager) RevokeContract(id string) error {
	contract, exists := cm.contracts[id]
	if !exists {
		return fmt.Errorf("contrato não encontrado: %s", id)
	}

	contract.Status = "revoked"
	return cm.saveContracts()
}

// AddTrigger adiciona um gatilho de execução ao contrato
func (cm *ContractManager) AddTrigger(contractID string, trigger Trigger) error {
	contract, exists := cm.contracts[contractID]
	if !exists {
		return fmt.Errorf("contrato não encontrado: %s", contractID)
	}

	contract.Triggers = append(contract.Triggers, trigger)
	return cm.saveContracts()
}
