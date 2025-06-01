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

	// Procura o diretório raiz do projeto (onde está o go.mod)
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("❌ Erro ao localizar o diretório do projeto: %v\n", err)
		os.Exit(1)
	}

	// Muda para o diretório raiz do projeto
	err = os.Chdir(projectRoot)
	if err != nil {
		fmt.Printf("❌ Erro ao mudar para o diretório do projeto: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📁 Executando testes em: %s\n", projectRoot)

	// Executa todos os testes recursivamente a partir do projeto
	cmd := exec.Command("go", "test", "./...", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("❌ Erro ao executar testes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Todos os testes concluídos!")
}

// findProjectRoot procura o diretório que contém o go.mod subindo a partir do diretório atual
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod não encontrado em nenhum diretório pai")
}
