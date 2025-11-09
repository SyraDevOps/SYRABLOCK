# 🚀 SYRABLOCK Terminal Unificado V10

## Visão Geral

O **SYRABLOCK Terminal Unificado** é uma aplicação de terminal interativa que integra todas as funcionalidades do sistema blockchain V10 em uma única interface estilizada e fácil de usar.

## ✨ Características Principais

### 🎨 Interface Estilizada
- Terminal com cores ANSI para melhor visualização
- Menus interativos e intuitivos
- Banner ASCII art personalizado
- Feedback visual para todas as operações

### 💼 Gerenciamento de Carteiras
- **Criar carteiras** com geração segura de endereços
- **Login** em carteiras existentes
- **Visualizar detalhes** completos da carteira
- **Gerar QR Codes** para compartilhamento
- **Verificação KYC** (simulada, integrável com serviços reais)
- **Listar todas as carteiras** disponíveis

### 💸 Sistema de Transações
- **Enviar SYRA** para outros endereços
- **Ver histórico** de transações da carteira
- **Transações pendentes** antes da mineração
- Assinatura criptográfica de transações

### ⛏️ Mineração
- **Minerar novos blocos** com sistema de recompensa
- **Ver status** de mineração pessoal
- Estatísticas de blocos minerados
- Hash rate e tempo de mineração em tempo real

### 🌐 Rede P2P
- **Conectar** à rede P2P
- **Ver status** da rede
- **Listar peers** conectados
- **Sincronizar blockchain** com a rede

### 📁 Gerenciamento de Arquivos
- **Registrar arquivos** na blockchain
- **Listar arquivos** registrados
- **Verificar integridade** de arquivos

### 🔗 Blockchain
- **Ver últimos blocos** minerados
- **Visualizar bloco específico** com detalhes
- **Validar integridade** da blockchain
- **Estatísticas** completas da blockchain

### ⚙️ Configurações
- **Token customizado** - Configure seu próprio token blockchain
- **Porta P2P** - Configure a porta de rede
- **Ver configurações** atuais
- **Resetar** configurações para o padrão

## 🎯 Configuração Personalizada na Primeira Inicialização

Na primeira vez que você executar o terminal, será solicitado a configurar:

1. **Token Customizado** - Define o nome do seu token (padrão: SYRA)
2. **Palavra de Busca** - Palavra que deve aparecer no hash dos blocos (padrão: Syra)
3. **Porta P2P** - Porta para conexões de rede (padrão: 8080)

Isso permite criar **blockchains personalizadas** com seus próprios parâmetros!

## 🚀 Como Usar

### Compilar

```bash
cd V10
go build -o syrablock_terminal cli_terminal.go
```

### Executar

```bash
./syrablock_terminal
```

Ou diretamente:

```bash
go run cli_terminal.go
```

### Gerar Executável para Distribuição

#### Linux
```bash
go build -o syrablock_terminal cli_terminal.go
```

#### Windows
```bash
GOOS=windows GOARCH=amd64 go build -o syrablock_terminal.exe cli_terminal.go
```

#### macOS
```bash
GOOS=darwin GOARCH=amd64 go build -o syrablock_terminal_macos cli_terminal.go
```

## 📋 Fluxo de Uso Típico

### 1. Primeira Inicialização
```
1. Execute o terminal
2. Configure token customizado (opcional)
3. Configure palavra de busca (opcional)
4. Configure porta P2P (opcional)
```

### 2. Criar e Configurar Carteira
```
1. Menu Principal → 1 (Carteiras)
2. Opção 1 (Criar Nova Carteira)
3. Digite um ID de usuário
4. Carteira criada automaticamente
5. Opção 4 (Gerar QR Code) - opcional
6. Opção 5 (Verificar KYC) - opcional
```

### 3. Minerar Blocos
```
1. Menu Principal → 3 (Mineração)
2. Opção 1 (Minerar Novo Bloco)
3. Aguarde a mineração (mostra progresso)
4. Receba recompensa de 50 SYRA
```

### 4. Enviar Transações
```
1. Menu Principal → 2 (Transações)
2. Opção 1 (Enviar SYRA)
3. Digite o endereço de destino
4. Digite a quantidade
5. Transação criada e adicionada ao pool
```

### 5. Ver Blockchain
```
1. Menu Principal → 6 (Blockchain)
2. Opção 1 (Ver Últimos Blocos)
3. Opção 3 (Validar Integridade)
4. Opção 4 (Estatísticas)
```

## 🔐 Segurança

- ✅ Assinaturas criptográficas SHA-256
- ✅ Chaves únicas por carteira
- ✅ Validação de integridade da blockchain
- ✅ Hashes seguros para todas as operações
- ✅ Armazenamento em arquivos JSON estruturados

## 📂 Estrutura de Dados

### Pastas Criadas
- `./data/` - Dados do sistema (transações pendentes, registros)
- `./PWtSY/` - Carteiras e QR codes
- `./tokens.json` - Blockchain principal

### Arquivos de Configuração
- `config.json` - Configurações do terminal

## 🛠️ Requisitos

- Go 1.24.3 ou superior
- Dependências:
  - `github.com/skip2/go-qrcode` - Para geração de QR codes

Instalar dependências:
```bash
go mod download
```

## 🎨 Menu Principal

```
═══════════════ MENU PRINCIPAL ═══════════════
1. 💼 Carteiras (Criar, Login, Gerenciar)
2. 💸 Transações (Enviar, Histórico)
3. ⛏️  Mineração (Minerar, Status)
4. 🌐 Rede P2P (Conectar, Status, Peers)
5. 📁 Arquivos (Registrar, Listar)
6. 🔗 Blockchain (Ver, Validar, Sincronizar)
7. ⚙️  Configurações (API, Token Customizado)
8. 🚪 Logout
9. ❌ Sair
═════════════════════════════════════════════
```

## 🌟 Funcionalidades Únicas

### Token Personalizado
Você pode criar sua própria blockchain personalizada alterando:
- Nome do token (ex: MYTOKEN, BITCOIN, etc)
- Palavra de busca no hash (ex: Custom, Mine, etc)

Isso é feito na primeira inicialização ou via menu de configurações.

### Sistema de Recompensas
- Cada bloco minerado: **50 SYRA**
- Transações incluídas nos blocos minerados
- Histórico completo de blocos registrados por carteira

### Registro de Arquivos
Registre qualquer arquivo na blockchain para:
- Prova de existência
- Verificação de integridade
- Timestamp imutável

## 🔄 Integração com Outros Módulos

Este terminal unifica os seguintes módulos V10:
- `PWtSY/wallet.go` - Sistema de carteiras
- `network/p2p_node.go` - Rede P2P
- `miner/` - Sistemas de mineração
- `transaction/` - Gerenciamento de transações
- `crypto/` - Criptografia e assinaturas

## 📊 Exemplo de Uso Completo

```bash
# 1. Compilar
go build -o syrablock_terminal cli_terminal.go

# 2. Executar
./syrablock_terminal

# 3. Configuração inicial (primeira vez)
Token customizado: MeuToken
Palavra de busca: Meu
Porta P2P: 8080

# 4. Criar carteira
Menu → 1 → 1
ID: Alice

# 5. Minerar alguns blocos
Menu → 3 → 1
(aguardar mineração)
Repetir 2-3 vezes

# 6. Ver saldo
Saldo aparece no menu principal: 150 SYRA (3 blocos * 50)

# 7. Enviar transação
Menu → 2 → 1
Destino: SYR1234567890abcdef...
Quantidade: 30

# 8. Ver blockchain
Menu → 6 → 1
(ver últimos blocos com suas transações)

# 9. Validar integridade
Menu → 6 → 3
✅ Blockchain íntegra!
```

## 🐛 Resolução de Problemas

### Erro ao criar carteira
- Verifique se a pasta `PWtSY` existe
- Verifique permissões de escrita

### Erro ao minerar
- Certifique-se de estar logado em uma carteira
- Verifique se `tokens.json` é acessível

### Erro de dependências
```bash
go mod tidy
go mod download
```

## 📝 Notas

- O sistema P2P é mostrado como "Simulado" - para funcionalidade completa, use os módulos P2P separados
- As transações são adicionadas aos blocos durante a mineração
- KYC é uma simulação - em produção, integre com serviço real
- Saldos são atualizados automaticamente após mineração

## 🎯 Próximos Passos

Para integração completa:
1. Integrar com `network/p2p_node.go` para rede real
2. Adicionar API REST para acesso externo
3. Implementar sincronização automática
4. Adicionar mais tipos de transações
5. Implementar contratos inteligentes no terminal

## 📞 Suporte

Este é o Terminal Unificado V10 do SYRABLOCK.
Para mais informações, consulte o `README.md` principal da pasta V10.

## ✅ Status

**100% Funcional e Pronto para Uso!**

Todas as funcionalidades básicas estão implementadas e testadas.
O terminal é totalmente interativo e pode ser compilado para executável standalone.

---

**Desenvolvido para facilitar a interatividade com o SYRABLOCK V10** 🚀
