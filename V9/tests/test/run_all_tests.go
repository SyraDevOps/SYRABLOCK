package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Este arquivo pode ser usado com "go run" para executar todos os testes
func main() {
	fmt.Println("🧪 Executando todos os testes PTW Blockchain...")

	// Obtém o diretório pai (onde estão os arquivos de teste)
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Erro ao obter diretório atual: %v\n", err)
		os.Exit(1)
	}

	parentDir := filepath.Dir(currentDir)

	// Muda para o diretório pai onde estão os testes
	err = os.Chdir(parentDir)
	if err != nil {
		fmt.Printf("❌ Erro ao mudar para o diretório dos testes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📁 Executando testes em: %s\n", parentDir)

	// Executa todos os testes no diretório dos testes
	cmd := exec.Command("go", "test", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("❌ Erro ao executar testes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Todos os testes concluídos!")
}
