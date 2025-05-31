package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Token struct {
	Index        int      `json:"index"`
	Nonce        int      `json:"nonce"`
	Hash         string   `json:"hash"`
	HashParts    []string `json:"hash_parts"`
	Timestamp    string   `json:"timestamp"`
	ContainsSyra bool     `json:"contains_syra"`
	Validator    string   `json:"validator,omitempty"`
	PrevHash     string   `json:"prev_hash,omitempty"` // NOVO CAMPO
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: go run validator.go <hash_do_bloco> <dono>")
		return
	}
	hash := os.Args[1]
	dono := os.Args[2]

	// Abre o arquivo tokens.json
	file, err := os.OpenFile("c:/Users/Syra_/Desktop/ptw/tokens.json", os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir tokens.json:", err)
		return
	}
	defer file.Close()

	var tokens []Token
	if err := json.NewDecoder(file).Decode(&tokens); err != nil {
		fmt.Println("Erro ao decodificar tokens.json:", err)
		return
	}

	// Procura o bloco pelo hash e adiciona o validator
	var blocoValidado *Token
	for i := range tokens {
		if tokens[i].Hash == hash {
			tokens[i].Validator = dono
			blocoValidado = &tokens[i]
			break
		}
	}

	if blocoValidado == nil {
		fmt.Println("Bloco não encontrado.")
		return
	}

	// Verifica a integridade da cadeia de blocos
	for i := 1; i < len(tokens); i++ {
		if tokens[i].PrevHash != tokens[i-1].Hash {
			fmt.Printf("Integridade quebrada no bloco %d!\n", tokens[i].Index)
			return
		}
	}

	// Salva o bloco validado em um novo arquivo
	out, err := os.Create("bloco_validado.json")
	if err != nil {
		fmt.Println("Erro ao criar bloco_validado.json:", err)
		return
	}
	defer out.Close()

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(blocoValidado); err != nil {
		fmt.Println("Erro ao salvar bloco_validado.json:", err)
		return
	}

	// Salva todos os tokens (com o validator atualizado) de volta no tokens.json
	file.Truncate(0)
	file.Seek(0, 0)
	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokens); err != nil {
		fmt.Println("Erro ao atualizar tokens.json:", err)
		return
	}

	fmt.Println("Bloco validado, salvo em bloco_validado.json e tokens.json atualizado!")
}
