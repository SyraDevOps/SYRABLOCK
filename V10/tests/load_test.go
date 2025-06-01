// filepath: c:\Users\Syra_\Desktop\ptw\tests\load_test.go
package tests

import (
	"fmt"
	"path/filepath" // Added for TestMiningPerformanceUnderLoad & TestMemoryUsageUnderLoad
	"runtime"       // For getCurrentMemoryUsage
	"sync"
	"testing"
	"time"
)

// --- Global Variables ---
var testNetworkGlobal *TestNetwork // Used by TestNode.BroadcastTransaction and TestNetwork methods

// --- Common Mock/Stub Type Definitions for 'tests' package ---

// Transaction represents a common mock transaction structure.
// Removed duplicate Transaction struct declaration

// Block represents a common mock block structure.
type Block struct {
	Hash         string
	Transactions []*Transaction // Slice of pointers to the common Transaction
	Timestamp    time.Time
	PrevHash     string
	Nonce        int
}

// Wallet represents a common mock wallet structure.
type Wallet struct {
	UserID      string
	Balance     int
	KYCVerified bool
}

// SaveWallet is a mock method for the Wallet.
// (Removed duplicate implementation to avoid redeclaration error)

// NewTransactionValidator creates a new mock transaction validator.
type TransactionValidator struct{}

func NewTransactionValidator() *TransactionValidator {
	// Implementação corrigida do TransactionValidator
	return &TransactionValidator{}
}

// Adicionar o método ValidateTransaction usado em integration_test.go
func (v *TransactionValidator) ValidateTransaction(tx *Transaction) bool {
	// Implementação simples para testes
	return tx != nil && tx.ID != ""
}

// MockTransactionPool simulates a transaction pool.
type MockTransactionPool struct {
	transactions map[string]*Transaction
	mutex        sync.Mutex
}

// Miner represents a mock miner.
type Miner struct {
	ID                  string
	BlockchainPath      string
	PendingTransactions []*Transaction
	blockchain          []*Block
	mutex               sync.Mutex
}

// ConsensusSystem represents a mock consensus system.
type ConsensusSystem struct {
	validators map[string]int // ID -> stake
	mutex      sync.Mutex
}

// ConsensusResult represents the result of consensus processing
type ConsensusResult struct {
	Approved  bool
	VoteCount int
}

// TestNode represents a node in the test network.
type TestNode struct {
	ID          int
	ReceivedTxs map[string]bool // txID -> received
	mutex       sync.Mutex
	// Fields required by recovery_test.go if TestNode is unified
	// These would be mock implementations
	GetBlockchainHeightFunc     func() (int, error)
	GetBlockHashAtHeightFunc    func(height int) (string, error)
	MineBlockFunc               func() (*Block, error)
	HasBlockFunc                func(hash string) bool
	BeginTransactionProcessFunc func(tx *Transaction) error
	GetTransactionStatusFunc    func(txID string) (string, error)
	VerifyTransactionResultFunc func(txID string) (bool, error)
	IsStateConsistentFunc       func() (bool, error)
}

// TestNetwork represents a mock P2P network.
type TestNetwork struct {
	Nodes []*TestNode
	mutex sync.Mutex
	// Fields/methods required by recovery_test.go if TestNetwork is unified
	CheckFullConnectivityFunc func() bool
	DisconnectNodesFunc       func(n1, n2 int)
	CheckConnectivityFunc     func(p1, p2 []int) bool
	ConnectNodesFunc          func(n1, n2 int)
}

// --- Helper Functions ---

// Transaction representa uma transação comum para todos os testes
type Transaction struct {
	ID        string
	From      string // Padronizado para From em vez de Sender
	To        string // Padronizado para To em vez de Receiver
	Amount    int
	Type      string
	Timestamp time.Time
	Data      string // Alterado para string em todos os locais
	Signature string
}

func createLoadTestTransaction(id string, userID int) *Transaction {
	return &Transaction{
		ID:        id,
		From:      fmt.Sprintf("User%d", userID),
		To:        fmt.Sprintf("Recipient%d", userID),
		Amount:    100,
		Type:      "load_test_transfer",
		Data:      fmt.Sprintf("data for %s from user %d", id, userID),
		Signature: "",
	}
}

func createTestBlock() *Block {
	numTx := 3
	txs := make([]*Transaction, numTx)
	for i := 0; i < numTx; i++ {
		txs[i] = createLoadTestTransaction(fmt.Sprintf("tx_in_block_%d_%d", time.Now().UnixNano(), i), i)
	}
	return &Block{
		Hash:         fmt.Sprintf("blockhash_%d", time.Now().UnixNano()),
		Transactions: txs,
		Timestamp:    time.Now(),
		PrevHash:     "genesis",
		Nonce:        0,
	}
}

func createTestNetwork(nodeCount int, blockchainPath string) *TestNetwork {
	nodes := make([]*TestNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = &TestNode{ID: i, ReceivedTxs: make(map[string]bool)}
	}
	network := &TestNetwork{Nodes: nodes}
	testNetworkGlobal = network // Assign to the global variable for simulation
	return network
}

func createRealConsensusSystem() *ConsensusSystem {
	return &ConsensusSystem{
		validators: make(map[string]int),
	}
}

func createRealMiner(id string, blockchainPath string) *Miner {
	return &Miner{
		ID:                  id,
		BlockchainPath:      blockchainPath,
		PendingTransactions: make([]*Transaction, 0),
		blockchain:          make([]*Block, 0),
	}
}

func getCurrentMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / (1024 * 1024) // Convert to MB
}

// --- Method Definitions for Mock/Stub Types ---

// MockTransactionPool methods
func (p *MockTransactionPool) AddTransaction(tx *Transaction) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if tx == nil || tx.ID == "" {
		return fmt.Errorf("invalid transaction")
	}
	if _, exists := p.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction %s already exists", tx.ID)
	}
	p.transactions[tx.ID] = tx
	return nil
}

// Miner methods
func (m *Miner) AddTransaction(tx *Transaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if tx == nil {
		return
	}
	m.PendingTransactions = append(m.PendingTransactions, tx)
}

func (m *Miner) ClearPendingTransactions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.PendingTransactions = make([]*Transaction, 0)
}

func (m *Miner) MineBlock(difficulty int) (*Block, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	newBlock := &Block{
		Hash:         fmt.Sprintf("mock-hash-%d-diff%d", time.Now().UnixNano(), difficulty), // Usa timestamp
		Transactions: make([]*Transaction, len(m.PendingTransactions)),
		Timestamp:    time.Now(),
		Nonce:        difficulty,
	}
	copy(newBlock.Transactions, m.PendingTransactions)

	if len(m.blockchain) > 0 {
		newBlock.PrevHash = m.blockchain[len(m.blockchain)-1].Hash
	} else {
		newBlock.PrevHash = "genesis_hash_0000000000000000"
	}

	m.blockchain = append(m.blockchain, newBlock)
	m.PendingTransactions = make([]*Transaction, 0)
	return newBlock, nil
}

// ConsensusSystem methods
func (cs *ConsensusSystem) RegisterValidator(id string, stake int) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.validators[id] = stake
}

func (cs *ConsensusSystem) ProcessBlock(block *Block) *ConsensusResult {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	approved := len(cs.validators) > 0 && block != nil && len(block.Transactions) >= 0 // Allow empty blocks
	return &ConsensusResult{
		Approved:  approved,
		VoteCount: len(cs.validators),
	}
}

// TestNode methods
func (n *TestNode) MineBlock() (*Block, error) {
	return &Block{
		Hash:      fmt.Sprintf("block-%d-%d", n.ID, time.Now().UnixNano()),
		Timestamp: time.Now(),
	}, nil
}

// TestNetwork methods
func (n *TestNetwork) Shutdown() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.Nodes = nil
	testNetworkGlobal = nil
}

func (n *TestNetwork) GetNode(id int) *TestNode {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.Nodes == nil || id < 0 || id >= len(n.Nodes) {
		return nil
	}
	return n.Nodes[id]
}

func (n *TestNetwork) CountNodesWithTransaction(txID string) int {
	count := 0
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.Nodes == nil {
		return 0
	}
	for _, node := range n.Nodes {
		node.mutex.Lock()
		if _, received := node.ReceivedTxs[txID]; received {
			count++
		}
		node.mutex.Unlock()
	}
	return count
}

// --- Test Functions ---

func TestTransactionPoolUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	pool := &MockTransactionPool{
		transactions: make(map[string]*Transaction),
	}

	var (
		totalTransactions     = 10000
		acceptedTransactions  = 0
		rejectedTransactions  = 0
		totalProcessingTimeMs int64
		statMutex             sync.Mutex // Renamed from mutex to avoid conflict
		wg                    sync.WaitGroup
	)

	concurrencyLevel := 50
	transactionsPerWorker := totalTransactions / concurrencyLevel

	startTime := time.Now()

	for i := 0; i < concurrencyLevel; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < transactionsPerWorker; j++ {
				txID := fmt.Sprintf("TX_LOAD_%d_%d", workerID, j)
				tx := createLoadTestTransaction(txID, workerID%10)

				txStartTime := time.Now()
				err := pool.AddTransaction(tx)
				processingTime := time.Since(txStartTime)

				statMutex.Lock()
				totalProcessingTimeMs += processingTime.Milliseconds()
				if err == nil {
					acceptedTransactions++
				} else {
					rejectedTransactions++
				}
				statMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	avgProcessingTime := 0.0
	if totalTransactions > 0 {
		avgProcessingTime = float64(totalProcessingTimeMs) / float64(totalTransactions)
	}

	tps := 0.0
	if totalTime.Seconds() > 0 {
		tps = float64(acceptedTransactions) / totalTime.Seconds()
	}

	t.Logf("Resultados do TestTransactionPoolUnderLoad:")
	t.Logf("  Total de transações para processar: %d", totalTransactions)
	t.Logf("  Transações aceitas: %d (%.1f%%)", acceptedTransactions,
		safePercentage(float64(acceptedTransactions), float64(totalTransactions)))
	t.Logf("  Transações rejeitadas: %d (%.1f%%)", rejectedTransactions,
		safePercentage(float64(rejectedTransactions), float64(totalTransactions)))
	t.Logf("  Tempo médio de processamento: %.2f ms", avgProcessingTime)
	t.Logf("  Taxa de transações por segundo (TPS): %.2f", tps)
	t.Logf("  Tempo total: %.2f segundos", totalTime.Seconds())

	if tps < 100 && totalTransactions > 0 {
		t.Logf("AVISO: Desempenho abaixo do esperado - TPS inferior a 100: %.2f", tps)
	}
}

func safePercentage(numerator, denominator float64) float64 {
	if denominator == 0 {
		if numerator == 0 {
			return 100.0
		}
		return 0.0
	}
	return (numerator / denominator) * 100.0
}

func TestMiningPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	tempBlockchainFile := filepath.Join(t.TempDir(), "load_test_blockchain.json")
	// No need for os.Remove, t.TempDir() handles cleanup.

	testMiner := createRealMiner("LoadTestMiner", tempBlockchainFile)
	blockSizes := []int{10, 50, 100, 200}

	for _, size := range blockSizes {
		t.Run(fmt.Sprintf("Block_%d_Tx", size), func(t *testing.T) {
			testMiner.ClearPendingTransactions()

			for i := 0; i < size; i++ {
				tx := createLoadTestTransaction(fmt.Sprintf("TX_BLOCK_%d_%d", size, i), i%20)
				testMiner.AddTransaction(tx) // tx is already *Transaction
			}

			startTime := time.Now()
			block, err := testMiner.MineBlock(1)
			miningTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Erro ao minerar bloco com %d transações: %v", size, err)
			}
			if block == nil {
				t.Fatalf("Bloco minerado é nulo para %d transações", size)
			}

			if len(block.Transactions) != size {
				t.Errorf("Bloco minerado tem %d transações, esperado: %d",
					len(block.Transactions), size)
			}

			txPerSec := 0.0
			if miningTime.Seconds() > 0 {
				txPerSec = float64(size) / miningTime.Seconds()
			}

			t.Logf("Bloco com %d transações minerado em %.2f segundos (%.2f tx/s)",
				size, miningTime.Seconds(), txPerSec)
		})
	}
}

func TestConsensusUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	validatorCounts := []int{10, 20, 50, 100}

	for _, validatorCount := range validatorCounts {
		t.Run(fmt.Sprintf("Validators_%d", validatorCount), func(t *testing.T) {
			consensus := createRealConsensusSystem()
			for i := 0; i < validatorCount; i++ {
				valID := fmt.Sprintf("Validator_%d", i)
				stake := 10 + (i % 90)
				consensus.RegisterValidator(valID, stake)
			}

			testBlock := createTestBlock() // Creates a block with some transactions

			startTime := time.Now()
			result := consensus.ProcessBlock(testBlock)
			consensusTime := time.Since(startTime)

			if !result.Approved {
				t.Errorf("Bloco não aprovado no consenso com %d validadores", validatorCount)
			}

			t.Logf("Consenso com %d validadores completado em %.2f segundos",
				validatorCount, consensusTime.Seconds())
			t.Logf("  Participação: %.1f%% (%d/%d)",
				safePercentage(float64(result.VoteCount), float64(validatorCount)),
				result.VoteCount, validatorCount)

			maxAllowedTimeFactor := 0.01 // 10ms per validator in mock
			maxAllowedTime := float64(validatorCount) * maxAllowedTimeFactor
			if consensusTime.Seconds() > maxAllowedTime && validatorCount > 0 && maxAllowedTime > 0 {
				t.Logf("AVISO: Consenso pode ser lento: %.2fs (máx sugerido: %.2fs)",
					consensusTime.Seconds(), maxAllowedTime)
			}
		})
	}
}

func TestNetworkPropagationUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	nodeCounts := []int{10, 20, 50}
	tempDir := t.TempDir() // t.TempDir() creates a unique temporary directory

	for _, nodeCount := range nodeCounts {
		t.Run(fmt.Sprintf("Nodes_%d", nodeCount), func(t *testing.T) {
			// Create a new network for each sub-test to ensure testNetworkGlobal is fresh
			currentTestNetwork := createTestNetwork(nodeCount, filepath.Join(tempDir, fmt.Sprintf("network_run_%d_nodes", nodeCount)))
			defer currentTestNetwork.Shutdown()

			if nodeCount == 0 {
				t.Log("Skipping test for 0 nodes.")
				return
			}
			sourceNode := currentTestNetwork.GetNode(0)
			if sourceNode == nil {
				t.Fatalf("Falha ao obter nó de origem (ID 0) para %d nós", nodeCount)
			}

			tx := createLoadTestTransaction("BROADCAST_TX_PROP", 1)

			startTime := time.Now()
			err := sourceNode.BroadcastTransaction(tx)
			if err != nil {
				t.Fatalf("Erro ao transmitir transação: %v", err)
			}

			maxWaitTime := 5 * time.Second
			allReceived := currentTestNetwork.WaitForPropagation(tx.ID, maxWaitTime)
			propagationTime := time.Since(startTime)

			nodesReached := currentTestNetwork.CountNodesWithTransaction(tx.ID)
			propagationPercentage := safePercentage(float64(nodesReached), float64(nodeCount))

			t.Logf("Propagação em rede com %d nós:", nodeCount)
			t.Logf("  Tempo de propagação: %.2f segundos", propagationTime.Seconds())
			t.Logf("  Nós alcançados: %d/%d (%.1f%%)",
				nodesReached, nodeCount, propagationPercentage)
			t.Logf("  Propagação completa (todos os nós receberam): %v", allReceived)

			if propagationPercentage < 95.0 && nodeCount > 1 {
				t.Errorf("Propagação insuficiente: %.1f%% (mínimo: 95%%)",
					propagationPercentage)
			}
			if !allReceived && nodeCount > 1 { // If not all received and there were nodes to receive
				t.Errorf("Propagação não foi completa para todos os nós.")
			}
		})
	}
}

func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	tempBlockchainFile := filepath.Join(t.TempDir(), "memory_test_blockchain.json")

	testMiner := createRealMiner("MemoryTestMiner", tempBlockchainFile)
	initialMemory := getCurrentMemoryUsage()

	blockCount := 5
	txPerBlock := 100

	for i := 0; i < blockCount; i++ {
		for j := 0; j < txPerBlock; j++ {
			tx := createLoadTestTransaction(fmt.Sprintf("TX_MEM_%d_%d", i, j), j%20)
			testMiner.AddTransaction(tx) // tx is already *Transaction
		}
		_, err := testMiner.MineBlock(1)
		if err != nil {
			t.Fatalf("Erro ao minerar bloco %d: %v", i, err)
		}
		if i > blockCount/2 {
			currentMemory := getCurrentMemoryUsage()
			if currentMemory > initialMemory*5 && initialMemory > 0.01 { // Check against significant growth
				t.Logf("AVISO: Uso de memória aumentou significativamente: %.2f MB -> %.2f MB após bloco %d",
					initialMemory, currentMemory, i+1)
			}
		}
	}

	finalMemory := getCurrentMemoryUsage()
	t.Logf("Uso de memória inicial: %.2f MB", initialMemory)
	t.Logf("Uso de memória final após %d blocos (%d tx cada): %.2f MB (aumento: %.2f MB)",
		blockCount, txPerBlock, finalMemory, finalMemory-initialMemory)
}

func TestBlockValidationUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de carga em modo curto")
	}

	numBlocksToValidate := 200
	concurrencyLevel := 10

	consensus := createRealConsensusSystem()
	for i := 0; i < 5; i++ {
		consensus.RegisterValidator(fmt.Sprintf("Val_%d", i), 100)
	}

	blocks := make([]*Block, numBlocksToValidate)
	for i := 0; i < numBlocksToValidate; i++ {
		b := createTestBlock() // createTestBlock already creates transactions
		b.Hash = fmt.Sprintf("block_to_validate_%d", i)
		blocks[i] = b
	}

	var wg sync.WaitGroup
	var validatedCount int
	var totalValidationTimeMs int64
	var validationMutex sync.Mutex

	blocksPerWorkerBase := numBlocksToValidate / concurrencyLevel
	remainderBlocks := numBlocksToValidate % concurrencyLevel

	processedBlocks := 0
	startTime := time.Now()

	for i := 0; i < concurrencyLevel; i++ {
		wg.Add(1)
		blocksForThisWorker := blocksPerWorkerBase
		if i < remainderBlocks {
			blocksForThisWorker++
		}

		startBlockIndex := processedBlocks
		endBlockIndex := processedBlocks + blocksForThisWorker
		processedBlocks = endBlockIndex

		if blocksForThisWorker == 0 {
			wg.Done()
			continue
		}

		go func(workerID int, currentWorkerBlocks []*Block) {
			defer wg.Done()
			for _, blockToValidate := range currentWorkerBlocks {
				valStartTime := time.Now()
				result := consensus.ProcessBlock(blockToValidate)
				valTime := time.Since(valStartTime)

				validationMutex.Lock()
				totalValidationTimeMs += valTime.Milliseconds()
				if result.Approved {
					validatedCount++
				}
				validationMutex.Unlock()
			}
		}(i, blocks[startBlockIndex:endBlockIndex])
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	avgValidationTime := 0.0
	if numBlocksToValidate > 0 {
		avgValidationTime = float64(totalValidationTimeMs) / float64(numBlocksToValidate)
	}

	blocksPerSecond := 0.0
	if totalDuration.Seconds() > 0 {
		blocksPerSecond = float64(validatedCount) / totalDuration.Seconds()
	}

	t.Logf("Resultados do TestBlockValidationUnderLoad:")
	t.Logf("  Total de blocos para validar: %d", numBlocksToValidate)
	t.Logf("  Blocos validados com sucesso: %d", validatedCount)
	t.Logf("  Nível de concorrência: %d", concurrencyLevel)
	t.Logf("  Tempo médio de validação por bloco: %.2f ms", avgValidationTime)
	t.Logf("  Taxa de validação de blocos por segundo (BPS): %.2f", blocksPerSecond)
	t.Logf("  Tempo total de validação: %.2f segundos", totalDuration.Seconds())

	if blocksPerSecond < 20 && numBlocksToValidate > 0 {
		t.Logf("AVISO: Desempenho de validação de blocos abaixo do esperado (BPS < 20): %.2f", blocksPerSecond)
	}
	if validatedCount != numBlocksToValidate && numBlocksToValidate > 0 {
		t.Errorf("Nem todos os blocos foram validados com sucesso. Esperado: %d, Obtido: %d", numBlocksToValidate, validatedCount)
	}
}
