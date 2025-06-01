package tests

import (
	"testing"
	"time"
)

// Estruturas de teste para auditoria
type TestAuditLog struct {
	ID        string
	Timestamp time.Time
	Action    string
	UserID    string
	Details   string
	Success   bool
	RiskLevel string
}

// Mock do sistema de auditoria
type mockAuditSystem struct {
	logs       []TestAuditLog
	alertCount int
	alertsById map[string]int
	violations map[string]int // UserID -> count
}

func newMockAuditSystem() *mockAuditSystem {
	return &mockAuditSystem{
		logs:       make([]TestAuditLog, 0),
		alertsById: make(map[string]int),
		violations: make(map[string]int),
	}
}

func (as *mockAuditSystem) logEvent(action, userID, details, riskLevel string, success bool) {
	log := TestAuditLog{
		ID:        "AUDIT_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		Action:    action,
		UserID:    userID,
		Details:   details,
		Success:   success,
		RiskLevel: riskLevel,
	}

	as.logs = append(as.logs, log)

	// Registra alertas para eventos críticos
	if riskLevel == "HIGH" || riskLevel == "CRITICAL" {
		as.alertCount++
		as.alertsById[userID]++
	}

	// Registra violações de segurança
	if !success && (action == "INVALID_SIGNATURE" || action == "UNAUTHORIZED_ACCESS") {
		as.violations[userID]++
	}
}

func (as *mockAuditSystem) getSecurityReport() map[string]interface{} {
	totalTransactions := 0
	failedTransactions := 0
	securityViolations := 0
	blocksMined := 0
	activeUsers := make(map[string]bool)

	for _, log := range as.logs {
		activeUsers[log.UserID] = true

		switch log.Action {
		case "TRANSACTION":
			totalTransactions++
			if !log.Success {
				failedTransactions++
			}
		case "BLOCK_MINED":
			blocksMined++
		case "INVALID_SIGNATURE", "UNAUTHORIZED_ACCESS":
			securityViolations++
		}
	}

	return map[string]interface{}{
		"total_transactions":  totalTransactions,
		"failed_transactions": failedTransactions,
		"security_violations": securityViolations,
		"blocks_mined":        blocksMined,
		"active_users":        len(activeUsers),
		"alerts":              as.alertCount,
		"critical_violations": len(as.violations),
	}
}

// Mock do sistema de segurança
type mockSecuritySystem struct {
	blacklist   map[string]time.Time   // IP -> quando expira
	rateLimiter map[string][]time.Time // IP -> timestamps das requisições
}

func newMockSecuritySystem() *mockSecuritySystem {
	return &mockSecuritySystem{
		blacklist:   make(map[string]time.Time),
		rateLimiter: make(map[string][]time.Time),
	}
}

func (ss *mockSecuritySystem) checkRateLimit(ip string, maxRequests int, window time.Duration) bool {
	now := time.Now()

	// Limpa requisições antigas
	if requests, exists := ss.rateLimiter[ip]; exists {
		validRequests := []time.Time{}
		for _, reqTime := range requests {
			if now.Sub(reqTime) < window {
				validRequests = append(validRequests, reqTime)
			}
		}
		ss.rateLimiter[ip] = validRequests
	}

	// Verifica se está na blacklist
	if expiry, blacklisted := ss.blacklist[ip]; blacklisted {
		if now.Before(expiry) {
			return false
		}
		delete(ss.blacklist, ip)
	}

	// Verifica limite
	if reqs, exists := ss.rateLimiter[ip]; exists && len(reqs) >= maxRequests {
		return false
	}

	// Adiciona nova requisição
	ss.rateLimiter[ip] = append(ss.rateLimiter[ip], now)
	return true
}

func (ss *mockSecuritySystem) blacklistIP(ip string, duration time.Duration) {
	ss.blacklist[ip] = time.Now().Add(duration)
}

// Testes para auditoria e segurança
func TestAuditLogging(t *testing.T) {
	as := newMockAuditSystem()

	// Testa log de evento normal
	as.logEvent("TRANSACTION", "Alice", "Transfer to Bob", "LOW", true)

	if len(as.logs) != 1 {
		t.Error("Falha ao registrar log de auditoria")
	}

	if as.alertCount != 0 {
		t.Error("Evento de baixo risco não deveria gerar alerta")
	}

	// Testa log de evento crítico
	as.logEvent("INVALID_SIGNATURE", "Eve", "Tentativa de falsificar assinatura", "CRITICAL", false)

	if as.alertCount != 1 {
		t.Error("Evento crítico deveria gerar alerta")
	}

	if as.violations["Eve"] != 1 {
		t.Error("Violação de segurança não foi registrada corretamente")
	}
}

func TestSecurityReport(t *testing.T) {
	as := newMockAuditSystem()

	// Popula logs de teste
	as.logEvent("TRANSACTION", "Alice", "Transfer to Bob", "LOW", true)
	as.logEvent("TRANSACTION", "Bob", "Transfer to Charlie", "LOW", true)
	as.logEvent("TRANSACTION", "Eve", "Transfer to Alice", "LOW", false)
	as.logEvent("BLOCK_MINED", "Charlie", "Block #1234", "LOW", true)
	as.logEvent("INVALID_SIGNATURE", "Eve", "Tentativa de falsificar assinatura", "CRITICAL", false)

	// Gera relatório
	report := as.getSecurityReport()

	// Verifica métricas no relatório
	if report["total_transactions"].(int) != 3 {
		t.Errorf("Contagem total de transações incorreta: %v", report["total_transactions"])
	}

	if report["failed_transactions"].(int) != 1 {
		t.Errorf("Contagem de transações falhas incorreta: %v", report["failed_transactions"])
	}

	if report["blocks_mined"].(int) != 1 {
		t.Errorf("Contagem de blocos minerados incorreta: %v", report["blocks_mined"])
	}

	// Corrigindo a expectativa para 4 usuários ativos (Alice, Bob, Charlie e Eve)
	if report["active_users"].(int) != 4 {
		t.Errorf("Contagem de usuários ativos incorreta: %v", report["active_users"])
	}

	if report["security_violations"].(int) != 1 {
		t.Errorf("Contagem de violações de segurança incorreta: %v", report["security_violations"])
	}
}

func TestRateLimiting(t *testing.T) {
	ss := newMockSecuritySystem()

	// Testa limite normal
	clientIP := "192.168.1.100"
	maxRequests := 5
	window := 1 * time.Minute

	// Primeiras 5 requisições devem passar
	for i := 0; i < maxRequests; i++ {
		if !ss.checkRateLimit(clientIP, maxRequests, window) {
			t.Errorf("Requisição %d deveria passar no rate limit", i+1)
		}
	}

	// A sexta requisição deve ser bloqueada
	if ss.checkRateLimit(clientIP, maxRequests, window) {
		t.Error("Rate limit não está funcionando, requisição excedente passou")
	}

	// Testa expiração da blacklist
	attackerIP := "10.0.0.5"
	ss.blacklistIP(attackerIP, 1*time.Millisecond)

	// Imediatamente após blacklist, deve ser bloqueado
	if ss.checkRateLimit(attackerIP, 10, window) {
		t.Error("IP na blacklist não foi bloqueado")
	}

	// Após expiração, deve permitir
	time.Sleep(2 * time.Millisecond)
	if !ss.checkRateLimit(attackerIP, 10, window) {
		t.Error("Blacklist não expirou quando deveria")
	}
}

func TestSecurityViolationsDetection(t *testing.T) {
	as := newMockAuditSystem()

	// Simula múltiplas tentativas de invasão do mesmo usuário
	for i := 0; i < 3; i++ {
		as.logEvent("INVALID_SIGNATURE", "Attacker", "Tentativa de falsificar assinatura", "HIGH", false)
	}

	if as.violations["Attacker"] != 3 {
		t.Error("Sistema não registrou múltiplas violações do mesmo usuário")
	}

	if as.alertCount != 3 {
		t.Error("Alertas não foram gerados para cada violação")
	}
}
