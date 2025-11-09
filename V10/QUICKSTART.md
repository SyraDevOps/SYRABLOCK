# 🚀 SYRABLOCK V10 - Guia de Início Rápido

## Instalação Rápida

### Opção 1: Usar Executável Pré-compilado

```bash
cd V10
./build/syrablock_terminal
```

### Opção 2: Compilar do Código Fonte

```bash
cd V10
go run cli_terminal.go
```

### Opção 3: Compilar para Múltiplas Plataformas

```bash
cd V10
./build.sh
```

Isso criará executáveis para:
- Linux: `build/syrablock_terminal`
- Windows: `build/syrablock_terminal.exe`
- macOS: `build/syrablock_terminal_macos`

## 📝 Primeira Execução

Na primeira vez que você executar o terminal, verá:

```
⚠️  Configuração não encontrada. Iniciando primeira configuração...

🚀 Bem-vindo ao SYRABLOCK Terminal!
═══════════════════════════════════

Esta é a primeira inicialização. Vamos configurar o sistema.

🔤 Token Customizado
Você pode definir um token personalizado para sua blockchain.
Deixe em branco para usar o padrão (SYRA).

Token customizado (ou Enter para SYRA): 
```

### Opções de Configuração

1. **Token Customizado** (opcional)
   - Exemplo: `BITCOIN`, `MYTOKEN`, `CUSTOM`
   - Padrão: `SYRA`

2. **Palavra de Busca** (opcional)
   - Palavra que deve aparecer no hash dos blocos
   - Exemplo: `Mine`, `Custom`, `Test`
   - Padrão: `Syra`

3. **Porta P2P** (opcional)
   - Porta para comunicação de rede
   - Exemplo: `8080`, `8333`, `9000`
   - Padrão: `8080`

**Dica**: Para usar os padrões, apenas pressione `Enter` em todas as opções.

## 🎮 Exemplo de Uso Completo

### Passo 1: Criar uma Carteira

```
Menu Principal → 1 (Carteiras)
→ 1 (Criar Nova Carteira)
→ Digite: Alice
→ [Enter]
```

**Resultado:**
```
✅ Carteira criada com sucesso!
👤 Usuário: Alice
📍 Endereço: SYR3f7a8b9c2d1e4f5g6h7i8j9k0l1m2n3
🔐 Assinatura: dGVzdF9zaWduYXR1cmU=...
```

### Passo 2: Minerar Alguns Blocos

```
Menu Principal → 3 (Mineração)
→ 1 (Minerar Novo Bloco)
→ Aguarde a mineração...
```

**Resultado:**
```
⛏️  Tentativas: 45673 | Tempo: 12.3s

✅ Bloco minerado com sucesso!
📦 Bloco: #1
🔑 Hash: dGVzdF9oYXNoXzEyMzQ1Njc4OTA...
🎲 Nonce: 45673
⏱️  Tempo: 12.34s
💰 Recompensa: 50 SYRA
```

**Dica**: Minere 2-3 blocos para ter saldo suficiente.

### Passo 3: Verificar Saldo

No menu principal, você verá:
```
👤 Usuário: Alice
💰 Saldo: 150 SYRA
📍 Endereço: SYR3f7a8b9c2d1e4f5...
```

### Passo 4: Criar Segunda Carteira (Bob)

```
Menu Principal → 1 (Carteiras)
→ 1 (Criar Nova Carteira)
→ Digite: Bob
→ [Enter]
```

### Passo 5: Voltar para Carteira da Alice

```
Menu Principal → 1 (Carteiras)
→ 2 (Login em Carteira Existente)
→ Digite: Alice
→ [Enter]
```

### Passo 6: Enviar SYRA para Bob

Primeiro, copie o endereço do Bob (Menu Carteiras → Login Bob → Ver Detalhes)

```
Menu Principal → 2 (Transações)
→ 1 (Enviar SYRA)
→ Endereço de destino: SYR... (endereço do Bob)
→ Quantidade: 30
→ [Enter]
```

**Resultado:**
```
✅ Transação criada com sucesso!
🆔 ID: a1b2c3d4e5f6g7h8
📍 Para: SYRf9e8d7c6b5a4...
💰 Quantidade: 30 SYRA
```

### Passo 7: Minerar para Confirmar a Transação

```
Menu Principal → 3 (Mineração)
→ 1 (Minerar Novo Bloco)
```

Agora a transação está incluída no bloco!

### Passo 8: Verificar Histórico de Transações

```
Menu Principal → 2 (Transações)
→ 2 (Ver Histórico de Transações)
```

**Resultado:**
```
#1
  ID: a1b2c3d4e5f6g7h8
  Tipo: transfer
  De: SYR3f7a8b9c2d1e4f5...
  Para: SYRf9e8d7c6b5a4...
  Valor: 30 SYRA
  Data: 09/11/2024 15:30
```

### Passo 9: Registrar um Arquivo

```
Menu Principal → 5 (Arquivos)
→ 1 (Registrar Arquivo na Blockchain)
→ Nome do arquivo: contrato.pdf
→ [Enter]
```

**Resultado:**
```
✅ Arquivo registrado com sucesso!
📄 Nome: contrato.pdf
🔑 Hash: 7d8e9f0a1b2c3d4e5f6g7h8i9j0k...
🆔 ID: x9y8z7w6v5u4t3
```

### Passo 10: Ver Blockchain

```
Menu Principal → 6 (Blockchain)
→ 1 (Ver Últimos Blocos)
```

**Resultado:**
```
#4
  Hash: gH8i9J0k1L2m3N4o5P6q...
  Minerador: Alice
  Transações: 1
  Data: 2024-11-09T15:35:00Z

#3
  Hash: aB1c2D3e4F5g6H7i8J9k...
  Minerador: Alice
  Transações: 0
  Data: 2024-11-09T15:30:00Z
...
```

### Passo 11: Validar Integridade

```
Menu Principal → 6 (Blockchain)
→ 3 (Validar Integridade da Blockchain)
```

**Resultado:**
```
✅ Blockchain íntegra! Todos os blocos validados.
```

### Passo 12: Ver Estatísticas

```
Menu Principal → 6 (Blockchain)
→ 4 (Estatísticas da Blockchain)
```

**Resultado:**
```
📦 Total de Blocos: 4
💸 Total de Transações: 1
⛏️  Mineradores Únicos: 1
📅 Primeiro Bloco: 2024-11-09T15:20:00Z
📅 Último Bloco: 2024-11-09T15:35:00Z
```

## 🎯 Atalhos e Dicas

### Navegação Rápida

- **Sempre pressione Enter** para confirmar ou continuar
- **Digite o número** da opção desejada
- **Menu Carteiras**: Opção `1` no menu principal
- **Menu Mineração**: Opção `3` no menu principal
- **Sair**: Opção `9` no menu principal

### Melhores Práticas

1. **Minere alguns blocos primeiro** para ter SYRA
2. **Crie QR Codes** das suas carteiras para backup
3. **Verifique KYC** para futuras funcionalidades
4. **Valide a blockchain** regularmente
5. **Faça backup** dos arquivos `.json` importantes

### Arquivos Importantes

- `config.json` - Suas configurações personalizadas
- `tokens.json` - Blockchain completa
- `PWtSY/wallet_*.json` - Suas carteiras
- `data/pending_transactions.json` - Transações pendentes
- `data/file_registry.json` - Arquivos registrados

## 🚨 Resolução de Problemas

### Erro: "Nenhuma carteira carregada"

**Solução**: Faça login primeiro (Menu 1 → Opção 2)

### Erro: "Saldo insuficiente"

**Solução**: Minere mais blocos para ganhar SYRA (Menu 3 → Opção 1)

### Erro: "Carteira já existe"

**Solução**: Use um ID diferente ou faça login na carteira existente

### Terminal não encontra arquivo

**Solução**: Execute sempre da pasta V10:
```bash
cd V10
./build/syrablock_terminal
```

### Erro de compilação

**Solução**: Verifique as dependências:
```bash
go mod download
go mod tidy
```

## 📚 Documentação Adicional

- **TERMINAL_README.md** - Documentação completa do terminal
- **README.md** - Visão geral do sistema V10
- **contracts/syrascript/README.md** - Linguagem de contratos

## 🆘 Comandos Úteis

```bash
# Ver versão do Go
go version

# Limpar builds anteriores
rm -rf build/

# Recompilar tudo
./build.sh

# Executar em modo verbose (debug)
go run cli_terminal.go

# Ver tamanho dos executáveis
ls -lh build/

# Fazer backup da blockchain
cp tokens.json tokens_backup_$(date +%Y%m%d).json

# Ver configuração atual
cat config.json
```

## 🎉 Pronto!

Você agora sabe como usar o SYRABLOCK Terminal Unificado!

Para mais informações, consulte:
- `TERMINAL_README.md` - Guia completo
- Menu de Ajuda dentro do terminal

**Divirta-se explorando sua blockchain personalizada!** 🚀
