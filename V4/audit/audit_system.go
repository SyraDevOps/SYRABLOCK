package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type AuditLog struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address,omitempty"`
	Success   bool      `json:"success"`
	Risk      string    `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
}

type SecurityMetrics struct {
	TotalTransactions  int `json:"total_transactions"`
	FailedTransactions int `json:"failed_transactions"`
	SecurityViolations int `json:"security_violations"`
	BlocksMined        int `json:"blocks_mined"`
	ActiveUsers        int `json:"active_users"`
}

func logSecurityEvent(action, userID, details, riskLevel string, success bool) {
	auditLog := AuditLog{
		ID:        fmt.Sprintf("AUDIT_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    action,
		UserID:    userID,
		Details:   details,
		Success:   success,
		Risk:      riskLevel,
	}

	// Log em arquivo JSON estruturado
	file, err := os.OpenFile("../security_audit.jsonl", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(auditLog)

	// Log crítico também em arquivo de texto
	if riskLevel == "CRITICAL" || riskLevel == "HIGH" {
		alertFile, err := os.OpenFile("../security_alerts.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			defer alertFile.Close()
			alertMsg := fmt.Sprintf("[%s] %s ALERT | User: %s | Action: %s | Details: %s\n",
				auditLog.Timestamp.Format(time.RFC3339), riskLevel, userID, action, details)
			alertFile.WriteString(alertMsg)
		}
	}
}

func generateSecurityReport() {
	fmt.Println("=== RELATÓRIO DE SEGURANÇA ===")

	// Lê logs de auditoria
	file, err := os.Open("../security_audit.jsonl")
	if err != nil {
		fmt.Println("Erro ao abrir arquivo de auditoria:", err)
		return
	}
	defer file.Close()

	var logs []AuditLog
	decoder := json.NewDecoder(file)
	for {
		var log AuditLog
		if err := decoder.Decode(&log); err != nil {
			break
		}
		logs = append(logs, log)
	}

	// Calcula métricas
	metrics := SecurityMetrics{}
	userMap := make(map[string]bool)

	for _, log := range logs {
		userMap[log.UserID] = true

		switch log.Action {
		case "TRANSACTION":
			metrics.TotalTransactions++
			if !log.Success {
				metrics.FailedTransactions++
			}
		case "BLOCK_MINED":
			if log.Success {
				metrics.BlocksMined++
			}
		}

		if log.Risk == "HIGH" || log.Risk == "CRITICAL" {
			metrics.SecurityViolations++
		}
	}

	metrics.ActiveUsers = len(userMap)

	fmt.Printf("Total de Transações: %d\n", metrics.TotalTransactions)
	fmt.Printf("Transações Falhadas: %d\n", metrics.FailedTransactions)
	fmt.Printf("Violações de Segurança: %d\n", metrics.SecurityViolations)
	fmt.Printf("Blocos Minerados: %d\n", metrics.BlocksMined)
	fmt.Printf("Usuários Ativos: %d\n", metrics.ActiveUsers)

	if metrics.TotalTransactions > 0 {
		successRate := float64(metrics.TotalTransactions-metrics.FailedTransactions) / float64(metrics.TotalTransactions) * 100
		fmt.Printf("Taxa de Sucesso: %.2f%%\n", successRate)
	}

	// Salva relatório
	reportFile, err := os.Create("../security_report.json")
	if err == nil {
		defer reportFile.Close()
		encoder := json.NewEncoder(reportFile)
		encoder.SetIndent("", "  ")
		encoder.Encode(metrics)
		fmt.Println("Relatório salvo em security_report.json")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run audit_system.go <comando>")
		fmt.Println("Comandos:")
		fmt.Println("  report - Gera relatório de segurança")
		fmt.Println("  test   - Teste do sistema de auditoria")
		return
	}

	switch os.Args[1] {
	case "report":
		generateSecurityReport()
	case "test":
		logSecurityEvent("TEST_TRANSACTION", "TestUser", "Teste de transação", "LOW", true)
		logSecurityEvent("SECURITY_VIOLATION", "MaliciousUser", "Tentativa de acesso não autorizado", "CRITICAL", false)
		fmt.Println("Eventos de teste logados")
	default:
		fmt.Println("Comando não reconhecido")
	}
}
