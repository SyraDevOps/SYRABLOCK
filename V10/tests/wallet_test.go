package tests

import (
	"testing"
)

// Estruturas de teste para carteiras
type TestWallet struct {
	UserID           string
	Address          string
	Balance          int
	RegisteredBlocks []string
	KYCVerified      bool
	Signature        string
}

// Mock do sistema de carteiras
type mockWalletSystemForWalletTest struct {
	wallets map[string]*TestWallet
}

func newMockWalletSystemForWalletTest() *mockWalletSystemForWalletTest {
	return &mockWalletSystemForWalletTest{
		wallets: make(map[string]*TestWallet),
	}
}

func (ws *mockWalletSystemForWalletTest) createWallet(userID string) *TestWallet {
	wallet := &TestWallet{
		UserID:           userID,
		Address:          "SYR" + userID + "123456789",
		Balance:          0,
		RegisteredBlocks: []string{},
		KYCVerified:      false,
		Signature:        "wallet-signature-" + userID,
	}
	ws.wallets[userID] = wallet
	return wallet
}

func (ws *mockWalletSystemForWalletTest) verifyKYC(userID string) bool {
	if wallet, exists := ws.wallets[userID]; exists {
		wallet.KYCVerified = true
		return true
	}
	return false
}

func (ws *mockWalletSystemForWalletTest) transfer(fromID, toID string, amount int) error {
	from, fromExists := ws.wallets[fromID]
	to, toExists := ws.wallets[toID]

	if !fromExists || !toExists {
		return &mockError{"Carteira não encontrada"}
	}

	if !from.KYCVerified || !to.KYCVerified {
		return &mockError{"Ambos usuários precisam ter KYC verificado"}
	}

	if from.Balance < amount {
		return &mockError{"Saldo insuficiente"}
	}

	from.Balance -= amount
	to.Balance += amount

	return nil
}

func (ws *mockWalletSystemForWalletTest) registerBlock(userID, blockHash string) error {
	wallet, exists := ws.wallets[userID]
	if !exists {
		return &mockError{"Carteira não encontrada"}
	}

	if !wallet.KYCVerified {
		return &mockError{"KYC não verificado"}
	}

	wallet.RegisteredBlocks = append(wallet.RegisteredBlocks, blockHash)
	wallet.Balance++ // Recompensa de mineração

	return nil
}

// Estrutura de erro para testes
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

// Testes para carteiras
func TestWalletCreation(t *testing.T) {
	ws := newMockWalletSystemForWalletTest()
	wallet := ws.createWallet("Alice")

	if wallet.UserID != "Alice" {
		t.Error("Criação de carteira falhou - UserID incorreto")
	}

	if wallet.Balance != 0 {
		t.Error("Nova carteira deveria ter saldo zero")
	}

	if wallet.KYCVerified {
		t.Error("Nova carteira não deveria ter KYC verificado")
	}
}

func TestKYCVerification(t *testing.T) {
	ws := newMockWalletSystemForWalletTest()
	wallet := ws.createWallet("Bob")

	if wallet.KYCVerified {
		t.Error("Carteira não deveria ter KYC verificado inicialmente")
	}

	success := ws.verifyKYC("Bob")
	if !success {
		t.Error("Verificação KYC falhou")
	}

	if !wallet.KYCVerified {
		t.Error("Carteira deveria ter KYC verificado após verifyKYC")
	}

	// Testa KYC para usuário inexistente
	success = ws.verifyKYC("NonExistentUser")
	if success {
		t.Error("Verificação KYC para usuário inexistente não deveria ter sucesso")
	}
}

func TestWalletTransfer(t *testing.T) {
	ws := newMockWalletSystemForWalletTest()

	// Cria e configura carteiras de teste
	alice := ws.createWallet("Alice")
	bob := ws.createWallet("Bob")
	alice.KYCVerified = true
	bob.KYCVerified = true
	alice.Balance = 100

	// Teste de transferência válida
	err := ws.transfer("Alice", "Bob", 30)
	if err != nil {
		t.Errorf("Transferência válida falhou: %v", err)
	}

	if alice.Balance != 70 || bob.Balance != 30 {
		t.Error("Saldos após transferência estão incorretos")
	}

	// Teste de transferência com saldo insuficiente
	err = ws.transfer("Alice", "Bob", 100)
	if err == nil {
		t.Error("Transferência com saldo insuficiente deveria falhar")
	}

	// Teste de transferência sem KYC
	charlie := ws.createWallet("Charlie")
	err = ws.transfer("Alice", "Charlie", 10)
	if charlie.KYCVerified {
		t.Error("Charlie não deveria ter KYC verificado")
	}
	if err == nil {
		t.Error("Transferência para usuário sem KYC deveria falhar")
	}
}

func TestRegisterMinedBlock(t *testing.T) {
	ws := newMockWalletSystemForWalletTest()

	// Cria carteira com KYC verificado
	alice := ws.createWallet("Alice")
	alice.KYCVerified = true

	// Teste de registro de bloco
	err := ws.registerBlock("Alice", "blockhash123")
	if err != nil {
		t.Errorf("Registro de bloco falhou: %v", err)
	}

	if len(alice.RegisteredBlocks) != 1 || alice.RegisteredBlocks[0] != "blockhash123" {
		t.Error("Bloco não foi registrado corretamente na carteira")
	}

	if alice.Balance != 1 {
		t.Error("Saldo da carteira deveria ser 1 após registro do bloco")
	}

	// Teste de registro sem KYC
	bob := ws.createWallet("Bob")
	if bob.KYCVerified {
		t.Error("Bob não deveria ter KYC verificado")
	}
	err = ws.registerBlock("Bob", "blockhash456")
	if err == nil {
		t.Error("Registro de bloco sem KYC deveria falhar")
	}
}
