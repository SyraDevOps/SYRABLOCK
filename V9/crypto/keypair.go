package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type KeyPair struct {
	UserID     string `json:"user_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	CreatedAt  string `json:"created_at"`
}

type DigitalSignature struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
	Timestamp string `json:"timestamp"`
}

func generateKeyPair(userID string) (*KeyPair, error) {
	// Gera par de chaves RSA 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Codifica chave privada
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Codifica chave pública
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Gera endereço baseado na chave pública
	hash := sha256.Sum256(publicKeyBytes)
	address := "SYRA" + base64.StdEncoding.EncodeToString(hash[:])[:32]

	keyPair := &KeyPair{
		UserID:     userID,
		PublicKey:  string(publicKeyPEM),
		PrivateKey: string(privateKeyPEM),
		Address:    address,
		CreatedAt:  fmt.Sprintf("%d", time.Now().Unix()),
	}

	return keyPair, nil
}

func saveKeyPair(keyPair *KeyPair) error {
	filename := filepath.Join("..", "PWtSY", fmt.Sprintf("keypair_%s.json", keyPair.UserID))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(keyPair)
}

func loadKeyPair(userID string) (*KeyPair, error) {
	filename := filepath.Join("..", "PWtSY", fmt.Sprintf("keypair_%s.json", userID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keyPair KeyPair
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&keyPair)
	return &keyPair, err
}

func signMessage(message string, privateKeyPEM string) (string, error) {
	// Decodifica chave privada
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("falha ao decodificar chave privada")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("não é uma chave RSA")
	}

	// Cria hash da mensagem
	hashed := sha256.Sum256([]byte(message))

	// Assina
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifySignature(message, signatureB64, publicKeyPEM string) bool {
	// Decodifica chave pública
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return false
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return false
	}

	// Decodifica assinatura
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false
	}

	// Verifica assinatura
	hashed := sha256.Sum256([]byte(message))
	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], signature)
	return err == nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run keypair.go <comando> [parametros]")
		fmt.Println("Comandos:")
		fmt.Println("  generate <user_id>                    - Gera novo par de chaves")
		fmt.Println("  sign <user_id> <message>              - Assina mensagem")
		fmt.Println("  verify <user_id> <message> <signature> - Verifica assinatura")
		return
	}

	switch os.Args[1] {
	case "generate":
		if len(os.Args) < 3 {
			fmt.Println("Erro: informe o user_id")
			return
		}
		userID := os.Args[2]

		keyPair, err := generateKeyPair(userID)
		if err != nil {
			fmt.Printf("Erro ao gerar chaves: %v\n", err)
			return
		}

		err = saveKeyPair(keyPair)
		if err != nil {
			fmt.Printf("Erro ao salvar chaves: %v\n", err)
			return
		}

		fmt.Printf("Par de chaves gerado para %s\n", userID)
		fmt.Printf("Endereço: %s\n", keyPair.Address)
		fmt.Printf("Chaves salvas em: keypair_%s.json\n", userID)

	case "sign":
		if len(os.Args) < 4 {
			fmt.Println("Erro: informe user_id e mensagem")
			return
		}
		userID := os.Args[2]
		message := os.Args[3]

		keyPair, err := loadKeyPair(userID)
		if err != nil {
			fmt.Printf("Erro ao carregar chaves: %v\n", err)
			return
		}

		signature, err := signMessage(message, keyPair.PrivateKey)
		if err != nil {
			fmt.Printf("Erro ao assinar: %v\n", err)
			return
		}

		fmt.Printf("Mensagem: %s\n", message)
		fmt.Printf("Assinatura: %s\n", signature)

	case "verify":
		if len(os.Args) < 5 {
			fmt.Println("Erro: informe user_id, mensagem e assinatura")
			return
		}
		userID := os.Args[2]
		message := os.Args[3]
		signature := os.Args[4]

		keyPair, err := loadKeyPair(userID)
		if err != nil {
			fmt.Printf("Erro ao carregar chaves: %v\n", err)
			return
		}

		valid := verifySignature(message, signature, keyPair.PublicKey)
		if valid {
			fmt.Println("✅ Assinatura VÁLIDA")
		} else {
			fmt.Println("❌ Assinatura INVÁLIDA")
		}

	default:
		fmt.Println("Comando não reconhecido")
	}
}
