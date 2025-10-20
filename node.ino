#include <ESP8266WiFi.h>
#include <ESP8266mDNS.h>
#include <ESP8266WebServer.h>
#include <ArduinoOTA.h>
#include <DNSServer.h>
#include <WiFiManager.h>
#include <ArduinoJson.h>
#include <FS.h>
#include <LittleFS.h>

const int botao = 0; // GPIO0 = botão FLASH
const int led = 2;   // LED interno (geralmente D4)

// Credenciais da rede que o ESP vai criar
const char* ap_ssid = "E-SYRA";
const char* ap_password = "SyraTeam";

// Nome do portal de configuração WiFi
const char* config_ap_ssid = "E-SYRA-Config";
const char* config_ap_password = "SyraTeam";

unsigned long tentativaInicio = 0;
const unsigned long tempoLimite = 20000; // 20 segundos para tentar conectar
bool modoAP = false;
bool modoManual = false; // controla se foi alterado manualmente
bool ultimoEstadoBotao = HIGH;
unsigned long ultimoTempoBotao = 0;
const unsigned long debounceDelay = 300;

// Controle do LED
unsigned long ultimoTempoLED = 0;
bool estadoLED = LOW;
unsigned long ultimoPrint = 0; // Para controlar o print dos pontos

// Dados do blockchain
String dadosBlockchain = "";

// Token único do dispositivo
String token = "";

// Status OTA
bool otaAtivo = false;
String otaStatus = "Inativo";

// Servidor HTTP
ESP8266WebServer server(80);

// DNS Server para captive portal
DNSServer dnsServer;
const byte DNS_PORT = 53;

// WiFiManager para configuração de rede
WiFiManager wifiManager;
bool wifiConfigMode = false;
String currentSSID = "";
String wifiStatus = "Desconectado";

// Reconexão WiFi automática
unsigned long ultimoTempoReconexao = 0;
const unsigned long intervaloReconexao = 15000; // 15 segundos

void setup() {
  pinMode(botao, INPUT_PULLUP);
  pinMode(led, OUTPUT);
  digitalWrite(led, HIGH); // LED apagado inicialmente
  Serial.begin(115200);
  delay(1000);
  
  // Inicializa sistema de arquivos
  if (!LittleFS.begin()) {
    Serial.println(F("❌ Erro ao inicializar LittleFS!"));
  } else {
    Serial.println(F("✅ LittleFS inicializado com sucesso!"));
  }
  
  Serial.println();
  Serial.println(F("========================================"));
  Serial.println(F("        Sistema SyraHome v1.0.0        "));
  Serial.println(F("========================================"));
  Serial.println(F("MÉTODOS DE ACESSO:"));
  Serial.println(F("• Modo Cliente: IP dinâmico + syra.local"));
  Serial.println(F("• Modo AP: http://192.168.4.1/ (PADRÃO)"));
  Serial.println(F("• Rede AP: E-SYRA / Senha: SyraTeam"));
  Serial.println(F("• Botão FLASH: Alterna entre modos"));
  Serial.println(F("========================================"));
  
  // Carrega ou cria dados do blockchain
  carregarDadosBlockchain();
  
  // Carrega ou cria token único do dispositivo
  carregarOuCriarToken();
  
  // Configura WiFiManager
  configurarWiFiManager();
  
  // Lê último modo salvo no LittleFS
  byte ultimoModo = lerModo();
  if (ultimoModo == 1) {
    Serial.println("Último modo: Access Point");
    criarRedeAP();
    modoManual = true;
  } else {
    Serial.println("Iniciando: Configuração automática de WiFi...");
    conectarWiFiManager();
  }
}

void loop() {
  // Verifica se o botão FLASH foi pressionado
  bool estadoBotao = digitalRead(botao);
  if (estadoBotao == LOW && ultimoEstadoBotao == HIGH && (millis() - ultimoTempoBotao > debounceDelay)) {
    alternarModo();
    ultimoTempoBotao = millis();
  }
  ultimoEstadoBotao = estadoBotao;

  if (!modoAP && WiFi.status() == WL_CONNECTED && !wifiConfigMode) {
    // Conectado com sucesso à rede WiFi
    static bool impresso = false;
    static bool mdnsIniciado = false;
    
    if (!impresso) {
      Serial.println();
      Serial.println("🌐 CONECTADO À WI-FI!");
      Serial.print("📡 IP do ESP: ");
      Serial.println(WiFi.localIP());
      Serial.print("📶 RSSI: ");
      Serial.print(WiFi.RSSI());
      Serial.println(" dBm");
      Serial.print("🌍 Rede: ");
      Serial.println(WiFi.SSID());
      currentSSID = WiFi.SSID();
      wifiStatus = "Conectado";
      Serial.println("🔵 LED: Piscando a cada 2 segundos");
      Serial.println("========================================");
      Serial.println("📱 ACESSO GARANTIDO POR IP: http://" + WiFi.localIP().toString() + "/");
      Serial.println("🌐 ACESSO POR mDNS (pode falhar): http://syra.local/");
      Serial.println("========================================");
      impresso = true;
    }
    
    // Inicia mDNS após conectar ao WiFi
    if (!mdnsIniciado) {
      // Tenta iniciar mDNS com retry
      bool mdnsOk = false;
      for (int tentativas = 0; tentativas < 3; tentativas++) {
        if (MDNS.begin("syra")) {
          mdnsOk = true;
          break;
        }
        delay(500);
      }
      
      if (mdnsOk) {
        Serial.println("🌍 mDNS iniciado com sucesso!");
        Serial.println("📱 Testando: http://syra.local/");
        MDNS.addService("http", "tcp", 80);
      } else {
        Serial.println("❌ Falha ao iniciar mDNS após 3 tentativas");
        Serial.println("⚠️ Use o IP diretamente: " + WiFi.localIP().toString());
      }
      
      // Inicia servidor independente do mDNS
      iniciarServidor();
      iniciarOTA();
      mdnsIniciado = true;
    }
    
    // Mantém o mDNS funcionando
    MDNS.update();
    // Processa requisições HTTP
    server.handleClient();
    // Processa OTA quando conectado
    ArduinoOTA.handle();
    
    // LED piscando a cada 2 segundos quando conectado ao WiFi
    controlarLEDWiFi();
    
  } else if (!modoAP && !modoManual && (millis() - tentativaInicio > tempoLimite)) {
    // Timeout automático - não conseguiu conectar, cria rede própria
    Serial.println();
    Serial.println("⏰ TIMEOUT: Não foi possível conectar à Wi-Fi.");
    Serial.println("🔄 Alternando para modo Access Point...");
    criarRedeAP();
    
  } else if (!modoAP && !modoManual) {
    // Ainda tentando conectar automaticamente
    if (millis() - ultimoPrint > 500) {
      Serial.print(".");
      ultimoPrint = millis();
    }
    // LED apagado enquanto tenta conectar
    digitalWrite(led, HIGH);
  }
  
  // Verificação de reconexão WiFi automática
  if (WiFi.status() != WL_CONNECTED && !modoAP && !wifiConfigMode) {
    if (millis() - ultimoTempoReconexao > intervaloReconexao) {
      Serial.println(F("🔄 Conexão Wi-Fi perdida. Tentando reconectar..."));
      WiFi.disconnect();
      WiFi.reconnect();
      wifiStatus = "Reconectando";
      ultimoTempoReconexao = millis();
      
      // LED pisca rapidamente durante reconexão
      digitalWrite(led, millis() % 300 < 150 ? LOW : HIGH);
    }
  }
  
  // Controla LED quando estiver em modo AP
  if (modoAP) {
    controlarLEDAP();
    // Mantém mDNS funcionando no modo AP também
    MDNS.update();
    // Processa requisições HTTP no modo AP
    server.handleClient();
    // Processa DNS para captive portal
    dnsServer.processNextRequest();
    // OTA também funciona em modo AP
    ArduinoOTA.handle();
  }
}

void alternarModo() {
  Serial.println();
  Serial.println("🔘 BOTÃO FLASH PRESSIONADO!");
  
  // Desconecta tudo antes de alternar
  WiFi.disconnect(true);
  delay(500);

  if (!modoAP) {
    // Muda para modo Access Point
    Serial.println("🔄 Mudando para modo ACCESS POINT...");
    criarRedeAP();
    // Salva modo no LittleFS
    salvarModo(1);
  } else {
    // Muda para modo cliente WiFi
    Serial.println("🔄 Mudando para modo CLIENTE WiFi...");
    // Limpa conexão WiFi antes de reconectar
    WiFi.disconnect(true);
    WiFi.mode(WIFI_OFF);
    delay(1000);
    
    // Usa WiFiManager para conectar
    conectarWiFiManager();
    modoAP = false;
    // Salva modo no LittleFS
    salvarModo(0);
  }

  modoManual = true;
}

void criarRedeAP() {
  WiFi.mode(WIFI_AP);
  delay(100);
  WiFi.softAP(ap_ssid, ap_password);
  
  IPAddress IP = WiFi.softAPIP();
  Serial.println();
  Serial.println("🏠 REDE ACCESS POINT CRIADA!");
  Serial.print("📡 Nome da rede: ");
  Serial.println(ap_ssid);
  Serial.print("🔐 Senha: ");
  Serial.println(ap_password);
  Serial.print("🌐 IP do ESP: ");
  Serial.println(IP);
  Serial.println("🔵 LED: Sempre ligado (modo AP)");
  Serial.println("========================================");
  Serial.println("📱 ACESSO PRINCIPAL: http://192.168.4.1/");
  Serial.println("🌐 CAPTIVE PORTAL: Redireciona automaticamente");
  Serial.println("⚡ ACESSO DIRETO: Funciona sempre no modo AP");
  Serial.println("========================================");
  
  // Inicia mDNS também no modo AP
  bool mdnsOk = false;
  for (int tentativas = 0; tentativas < 3; tentativas++) {
    if (MDNS.begin("syra")) {
      mdnsOk = true;
      break;
    }
    delay(500);
  }
  
  if (mdnsOk) {
    Serial.println("🌍 mDNS AP iniciado (pode não funcionar)!");
    Serial.println("� Alternativo mDNS: http://syra.local/");
    MDNS.addService("http", "tcp", 80);
  } else {
    Serial.println("❌ mDNS falhou no modo AP (normal)");
  }
  
  // Configura captive portal
  dnsServer.start(DNS_PORT, "*", IP);
  Serial.println("🌐 Captive Portal ativo - redirecionamento automático!");
  
  // Inicia servidor independente do mDNS
  iniciarServidor();
  iniciarOTA();
  
  Serial.println("========================================");
  
  modoAP = true;
}

void controlarLEDWiFi() {
  // Pisca a cada 2 segundos quando conectado ao WiFi
  if (millis() - ultimoTempoLED > 2000) {
    estadoLED = !estadoLED;
    digitalWrite(led, estadoLED);
    ultimoTempoLED = millis();
  }
}

void controlarLEDAP() {
  // LED sempre ligado quando em modo Access Point
  digitalWrite(led, LOW);
}

void salvarDadosBlockchain(String dados) {
  File f = LittleFS.open("/blockchain.txt", "w");
  if (!f) {
    Serial.println("❌ Erro ao abrir arquivo para escrita!");
    return;
  }
  f.print(dados);
  f.close();
  Serial.println("💾 Dados do blockchain salvos!");
}

String lerDadosBlockchain() {
  File f = LittleFS.open("/blockchain.txt", "r");
  if (!f) {
    Serial.println("📄 Nenhum dado de blockchain salvo.");
    return "";
  }
  String dados = f.readString();
  f.close();
  return dados;
}

void carregarDadosBlockchain() {
  dadosBlockchain = lerDadosBlockchain();
  
  if (dadosBlockchain == "") {
    // Cria dados iniciais do blockchain
    dadosBlockchain = "{\n";
    dadosBlockchain += "  \"index\": 1,\n";
    dadosBlockchain += "  \"nonce\": 488889,\n";
    dadosBlockchain += "  \"hash\": \"bIGfcUj9LSyraaTMGBINCSdLzaUVk+1DV8RKSVUYIqc=\",\n";
    dadosBlockchain += "  \"hash_parts\": [\n";
    dadosBlockchain += "    \"phVPqolgdP3PzRzJdJfe17N5L2ttP8ISeU1MIY7mFwk=\",\n";
    dadosBlockchain += "    \"nzhDLsoKEn1Qjo94V6kw6GnqA26tLwK2J8AnIVY2gTg=\",\n";
    dadosBlockchain += "    \"56tQ53tc2B2fV78zhyEACht7CFGWwQuYcuWhqECQtMY=\",\n";
    dadosBlockchain += "    \"NRrcCCR4gCUUcQ7P/eNB9PGfTpOwDe2ObeJbVK+sKB8=\"\n";
    dadosBlockchain += "  ],\n";
    dadosBlockchain += "  \"timestamp\": \"2025-05-31T17:21:03-03:00\",\n";
    dadosBlockchain += "  \"contains_syra\": true\n";
    dadosBlockchain += "}";
    
    salvarDadosBlockchain(dadosBlockchain);
    Serial.println("🔗 Dados iniciais do blockchain criados!");
  } else {
    Serial.println("🔗 Dados do blockchain carregados da memória!");
  }
  
  Serial.println("📋 Blockchain atual:");
  Serial.println(dadosBlockchain);
  Serial.println("========================================");
}

void carregarOuCriarToken() {
  File f = LittleFS.open("/token.txt", "r");
  if (!f) {
    // Cria token único baseado no chip ID
    token = "SYRA-" + String(ESP.getChipId(), HEX);
    token.toUpperCase(); // Garante que seja maiúsculo
    
    File fw = LittleFS.open("/token.txt", "w");
    if (fw) {
      fw.print(token);
      fw.close();
      Serial.println("🆕 Token gerado e salvo!");
    } else {
      Serial.println("❌ Erro ao salvar token!");
    }
  } else {
    token = f.readString();
    token.trim(); // Remove espaços e quebras de linha
    f.close();
    Serial.println("🔑 Token carregado da memória!");
  }
  
  Serial.print("🏷️  Token do dispositivo: ");
  Serial.println(token);
  Serial.print("🔧 Chip ID: ");
  Serial.println(String(ESP.getChipId(), HEX));
  Serial.println("========================================");
}

void iniciarServidor() {
  // Rota principal - Interface HTML simples
  server.on("/", []() {
    String html = "<!DOCTYPE html><html><head>";
    html += "<meta charset='UTF-8'>";
    html += "<meta name='viewport' content='width=device-width, initial-scale=1.0'>";
    html += "<title>SyraHome - Controle</title>";
    html += "</head><body>";
    html += "<h1>SyraHome v1.0.0</h1>";
    html += "<hr>";
    
    // Informações do dispositivo
    html += "<h2>Informações do Dispositivo</h2>";
    html += "<p><strong>Token:</strong> " + token + "</p>";
    html += "<p><strong>Chip ID:</strong> " + String(ESP.getChipId(), HEX) + "</p>";
    html += "<p><strong>MAC:</strong> " + WiFi.macAddress() + "</p>";
    html += "<p><strong>IP:</strong> " + (modoAP ? WiFi.softAPIP().toString() : WiFi.localIP().toString()) + "</p>";
    html += "<p><strong>Modo:</strong> " + String(modoAP ? "Access Point" : "Cliente WiFi") + "</p>";
    html += "<p><strong>Memória Livre:</strong> " + String(ESP.getFreeHeap()) + " bytes</p>";
    html += "<p><strong>Uptime:</strong> " + String(millis() / 1000) + " segundos</p>";
    
    if (!modoAP) {
      html += "<p><strong>Rede:</strong> " + (currentSSID != "" ? currentSSID : "N/A") + "</p>";
      html += "<p><strong>Sinal:</strong> " + String(WiFi.RSSI()) + " dBm</p>";
    } else {
      html += "<p><strong>Clientes:</strong> " + String(WiFi.softAPgetStationNum()) + "</p>";
    }
    
    html += "<hr>";
    
    // Controles
    html += "<h2>Controles</h2>";
    html += "<form method='POST' action='/modo'>";
    if (modoAP) {
      html += "<button type='submit' name='acao' value='online'>Conectar à WiFi (Online)</button>";
    } else {
      html += "<button type='submit' name='acao' value='offline'>Modo Offline (Access Point)</button>";
    }
    html += "</form>";
    html += "<br>";
    
    html += "<form method='POST' action='/config'>";
    html += "<button type='submit'>Configurar Rede WiFi</button>";
    html += "</form>";
    
    html += "<hr>";
    
    // APIs disponíveis
    html += "<h2>APIs Disponíveis</h2>";
    html += "<ul>";
    html += "<li><a href='/status'>Status Completo (JSON)</a></li>";
    html += "<li><a href='/token'>Token do Dispositivo (JSON)</a></li>";
    html += "<li><a href='/versao'>Versão do Sistema (JSON)</a></li>";
    html += "<li><a href='/blockchain'>Dados Blockchain (JSON)</a></li>";
    html += "<li><a href='/ota'>Atualização OTA (JSON)</a></li>";
    html += "<li><a href='/video/1'>Holo-teste 1</a></li>";
    html += "<li><a href='/video/2'>Holo-teste 2</a></li>";
    html += "</ul>";
    
    html += "<hr>";
    html += "<p><small>SyraHome IoT System - " + String(ESP.getChipId(), HEX) + "</small></p>";
    html += "</body></html>";
    
    server.send(200, "text/html", html);
  });

  // Rota para dados blockchain
  server.on("/blockchain", []() {
    DynamicJsonDocument doc(1024);
    
    // Lê dados existentes
    String dadosBlockchain = lerDadosBlockchain();
    
    doc["sistema"] = "SyraHome Blockchain";
    doc["versao_protocolo"] = "1.0";
    doc["total_transacoes"] = dadosBlockchain.length() > 0 ? 1 : 0;
    doc["ultimo_hash"] = String(ESP.getChipId(), HEX);
    doc["timestamp"] = millis();
    doc["node_id"] = token;
    doc["network"] = "SyraNet";
    
    // Dados da transação atual
    JsonObject transacao = doc.createNestedObject("transacao_atual");
    transacao["id"] = String(ESP.getChipId(), HEX) + String(millis());
    transacao["tipo"] = "device_status";
    transacao["dados"] = dadosBlockchain.length() > 0 ? dadosBlockchain : "Nenhum dado blockchain disponível";
    transacao["validado"] = true;
    transacao["timestamp"] = millis();
    
    // Estatísticas da rede
    JsonObject stats = doc.createNestedObject("estatisticas");
    stats["nodes_ativos"] = 1;
    stats["uptime"] = millis();
    stats["memoria_utilizada"] = ESP.getFlashChipSize() - ESP.getFreeHeap();
    stats["conectividade"] = WiFi.status() == WL_CONNECTED ? "online" : "offline";
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Rota para status do sistema
  server.on("/status", []() {
    DynamicJsonDocument doc(1024);
    
    doc["sistema"] = "SyraHome v1.0.0";
    doc["modo"] = modoAP ? "Access Point" : "Cliente WiFi";
    doc["ip"] = modoAP ? WiFi.softAPIP().toString() : WiFi.localIP().toString();
    doc["mac"] = WiFi.macAddress();
    doc["uptime"] = millis();
    doc["memoria_livre"] = ESP.getFreeHeap();
    doc["chip_id"] = String(ESP.getChipId(), HEX);
    doc["token"] = token;
    doc["ota_ativo"] = otaAtivo;
    doc["ota_status"] = otaStatus;
    
    if (!modoAP) {
      doc["rssi"] = WiFi.RSSI();
      doc["rede_wifi"] = currentSSID != "" ? currentSSID : "N/A";
      doc["wifi_status"] = wifiStatus;
    } else {
      doc["clientes_ap"] = WiFi.softAPgetStationNum();
      doc["rede_ap"] = ap_ssid;
    }
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Rota para versão do sistema
  server.on("/versao", []() {
    DynamicJsonDocument doc(512);
    
    doc["sistema"] = "SyraHome";
    doc["versao"] = "1.0.0";
    doc["build"] = "20251020";
    doc["desenvolvedor"] = "SyraDevOps";
    doc["plataforma"] = "ESP8266";
    doc["memoria_total"] = ESP.getFlashChipSize();
    doc["memoria_livre"] = ESP.getFreeHeap();
    doc["chip_id"] = String(ESP.getChipId(), HEX);
    doc["sdk_version"] = ESP.getSdkVersion();
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Rota para informações do token
  server.on("/token", []() {
    DynamicJsonDocument doc(512);
    
    doc["token"] = token;
    doc["chip_id"] = String(ESP.getChipId(), HEX);
    doc["mac_address"] = WiFi.macAddress();
    doc["sistema"] = "SyraHome v1.0.0";
    doc["gerado_em"] = "Primeira inicialização";
    doc["persistente"] = true;
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Rotas para vídeos Holo-testes
  server.on("/video/1", []() {
    DynamicJsonDocument doc(512);
    
    doc["titulo"] = "Holo-teste 1";
    doc["descricao"] = "Demonstração de funcionalidades básicas do SyraHome";
    doc["url_redirect"] = "https://www.youtube.com/watch?v=B30S0Vr9N9A&t=1060s";
    doc["duracao"] = "5:30";
    doc["tipo"] = "demonstracao_basica";
    doc["status"] = "ativo";
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.sendHeader("Location", "https://www.youtube.com/watch?v=B30S0Vr9N9A&t=1060s");
    server.send(302, "application/json", output);
  });
  
  server.on("/video/2", []() {
    DynamicJsonDocument doc(512);
    
    doc["titulo"] = "Holo-teste 2";
    doc["descricao"] = "Funcionalidades avançadas e integração IoT";
    doc["url_redirect"] = "https://www.youtube.com/watch?v=UJoggmMOAEk&t=2075s";
    doc["duracao"] = "8:45";
    doc["tipo"] = "demonstracao_avancada";
    doc["status"] = "ativo";
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.sendHeader("Location", "https://www.youtube.com/watch?v=UJoggmMOAEk&t=2075s");
    server.send(302, "application/json", output);
  });

  // Rota para OTA
  server.on("/ota", []() {
    DynamicJsonDocument doc(1024);
    
    doc["sistema"] = "SyraHome OTA";
    doc["status"] = otaStatus;
    doc["ativo"] = otaAtivo;
    doc["hostname"] = "syra.local";
    doc["porta"] = 8266;
    doc["senha_requerida"] = true;
    doc["versao_atual"] = "SyraHome v1.0.0";
    doc["build_atual"] = "20251020";
    doc["uptime"] = millis();
    doc["memoria_livre"] = ESP.getFreeHeap();
    doc["chip_id"] = String(ESP.getChipId(), HEX);
    doc["sdk_version"] = ESP.getSdkVersion();
    
    JsonObject instrucoes = doc.createNestedObject("instrucoes");
    instrucoes["arduino_ide"] = "Ferramentas → Porta → syra.local";
    instrucoes["upload"] = "Sketch → Carregar (Ctrl+U)";
    instrucoes["aguardar"] = "Aguarde a conclusão da atualização";
    
    JsonObject aviso = doc.createNestedObject("aviso");
    aviso["importante"] = "Não desligue o ESP durante a atualização";
    aviso["senha"] = "Use a senha: SyraTeam";
    aviso["rede"] = "Dispositivo deve estar conectado à rede";
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Rota para alternar modo (offline/online)
  server.on("/modo", HTTP_POST, []() {
    String acao = server.arg("acao");
    
    if (acao == "offline" && !modoAP) {
      // Muda para modo Access Point (offline)
      Serial.println(F("🔄 Comando web: Mudando para modo offline (AP)"));
      WiFi.disconnect(true);
      delay(500);
      criarRedeAP();
      salvarModo(1);
      modoManual = true;
    } else if (acao == "online" && modoAP) {
      // Muda para modo cliente WiFi (online)
      Serial.println(F("🔄 Comando web: Mudando para modo online (Cliente)"));
      WiFi.disconnect(true);
      WiFi.mode(WIFI_OFF);
      delay(1000);
      conectarWiFiManager();
      modoAP = false;
      salvarModo(0);
      modoManual = true;
    }
    
    // Redireciona de volta para a página principal
    server.sendHeader("Location", "/");
    server.send(302, "text/plain", "Redirecionando...");
  });

  // Rota para configuração de rede
  server.on("/config", HTTP_POST, []() {
    Serial.println(F("🔧 Comando web: Iniciando configuração WiFi"));
    
    // Reset das configurações salvas do WiFiManager
    wifiManager.resetSettings();
    
    // Força o modo de configuração
    WiFi.disconnect(true);
    WiFi.mode(WIFI_OFF);
    delay(1000);
    
    // Inicia o portal de configuração
    if (!wifiManager.startConfigPortal(config_ap_ssid, config_ap_password)) {
      Serial.println(F("❌ Falha ao iniciar portal de configuração"));
      server.send(500, "text/html", "<html><body><h1>Erro ao iniciar configuração</h1><a href='/'>Voltar</a></body></html>");
    } else {
      Serial.println(F("✅ Portal de configuração iniciado"));
      server.send(200, "text/html", "<html><body><h1>Portal de Configuração Ativo</h1><p>Conecte-se à rede E-SYRA-Config para configurar</p><a href='/'>Voltar</a></body></html>");
    }
  });

  // Rota API para compatibilidade (retorna JSON)
  server.on("/api", []() {
    DynamicJsonDocument doc(1024);
    
    doc["sistema"] = "SyraHome v1.0.0";
    doc["descricao"] = "Sistema IoT com portal captivo e gerenciamento de rede";
    
    // Informações de acesso
    JsonObject acesso = doc.createNestedObject("metodos_acesso");
    if (modoAP) {
      acesso["ip_principal"] = "http://192.168.4.1/";
      acesso["rede_wifi"] = "E-SYRA";
      acesso["senha_wifi"] = "SyraTeam";
      acesso["captive_portal"] = "Redirecionamento automático";
      acesso["mdns_alternativo"] = "http://syra.local/ (pode falhar)";
    } else {
      acesso["ip_principal"] = "http://" + WiFi.localIP().toString() + "/";
      acesso["mdns_alternativo"] = "http://syra.local/ (pode falhar)";
      acesso["rede_conectada"] = currentSSID;
    }
    
    doc["rotas_disponiveis"] = JsonArray();
    
    JsonArray rotas = doc["rotas_disponiveis"];
    rotas.add("/status - Status do sistema");
    rotas.add("/blockchain - Dados blockchain");
    rotas.add("/token - Informações do token");
    rotas.add("/versao - Versão do sistema");
    rotas.add("/video/1 - Redirecionamento Holo-teste 1");
    rotas.add("/video/2 - Redirecionamento Holo-teste 2");
    rotas.add("/ota - Atualização OTA");
    
    doc["modo_atual"] = modoAP ? "Access Point" : "Cliente WiFi";
    doc["ip"] = modoAP ? WiFi.softAPIP().toString() : WiFi.localIP().toString();
    doc["memoria_livre"] = ESP.getFreeHeap();
    doc["token"] = token;
    doc["uptime"] = millis();
    
    if (!modoAP) {
      doc["rssi"] = WiFi.RSSI();
      doc["rede_conectada"] = currentSSID != "" ? currentSSID : "N/A";
    } else {
      doc["clientes_conectados"] = WiFi.softAPgetStationNum();
      doc["rede_ap"] = ap_ssid;
    }
    
    String output;
    serializeJson(doc, output);
    
    server.sendHeader("Content-Type", "application/json");
    server.sendHeader("Access-Control-Allow-Origin", "*");
    server.send(200, "application/json", output);
  });

  // Captive portal - redireciona tudo para API principal
  server.onNotFound([]() {
    String requestedHost = server.hostHeader();
    String currentIP = modoAP ? WiFi.softAPIP().toString() : WiFi.localIP().toString();
    
    // Se está em modo AP e não é o IP correto, redireciona
    if (modoAP && requestedHost != currentIP && requestedHost != "syra.local") {
      Serial.println(F("🔄 Captive Portal: redirecionando ") + server.client().remoteIP().toString() + F(" de ") + requestedHost);
      
      DynamicJsonDocument doc(512);
      
      doc["captive_portal"] = true;
      doc["redirect_to"] = "http://syra.local/";
      doc["sistema"] = "SyraHome v1.0.0";
      doc["mensagem"] = "Redirecionamento automático para portal principal";
      doc["rotas_disponiveis"] = JsonArray();
      
      JsonArray rotas = doc["rotas_disponiveis"];
      rotas.add("/status");
      rotas.add("/blockchain");
      rotas.add("/token");
      rotas.add("/versao");
      rotas.add("/video/1");
      rotas.add("/video/2");
      
      String output;
      serializeJson(doc, output);
      
      server.sendHeader("Content-Type", "application/json");
      server.sendHeader("Access-Control-Allow-Origin", "*");
      server.sendHeader("Location", "http://syra.local/");
      server.send(302, "application/json", output);
    } else {
      // Página 404 em JSON
      DynamicJsonDocument doc(256);
      doc["erro"] = "Página não encontrada";
      doc["codigo"] = 404;
      doc["sistema"] = "SyraHome v1.0.0";
      doc["sugestao"] = "Acesse / para ver as rotas disponíveis";
      
      String output;
      serializeJson(doc, output);
      
      server.sendHeader("Content-Type", "application/json");
      server.send(404, "application/json", output);
    }
  });

  server.begin();
  
  // Mostra URLs com IP e mDNS
  String currentIP = modoAP ? WiFi.softAPIP().toString() : WiFi.localIP().toString();
  
  Serial.println("🛰️ Servidor HTTP iniciado!");
  Serial.println("========================================");
  Serial.println("📍 ACESSO PRINCIPAL (SEMPRE FUNCIONA):");
  Serial.println("   http://" + currentIP + "/");
  Serial.println("🌐 ACESSO ALTERNATIVO (pode falhar):");
  Serial.println("   http://syra.local/");
  if (modoAP) {
    Serial.println("📱 MODO ACCESS POINT ATIVO:");
    Serial.println("   Rede: E-SYRA | Senha: SyraTeam");
    Serial.println("   IP Padrão: 192.168.4.1");
  }
  Serial.println("========================================");
  Serial.println("📦 Blockchain: http://" + currentIP + "/blockchain");
  Serial.println("📊 Status: http://" + currentIP + "/status");
  Serial.println("🏷️ Token: http://" + currentIP + "/token");
  Serial.println("📱 Versão: http://" + currentIP + "/versao");
  Serial.println("🎥 Vídeo 1: http://" + currentIP + "/video/1");
  Serial.println("🎥 Vídeo 2: http://" + currentIP + "/video/2");
  Serial.println("🚀 OTA: http://" + currentIP + "/ota");
  Serial.println("========================================");
}

void iniciarOTA() {
  ArduinoOTA.setHostname("syra");
  ArduinoOTA.setPassword("SyraTeam"); // Mesma senha do AP
  
  ArduinoOTA.onStart([]() {
    String tipo = (ArduinoOTA.getCommand() == U_FLASH) ? "sketch" : "filesystem";
    Serial.println("🚀 Iniciando OTA (" + tipo + ")...");
    otaAtivo = true;
    otaStatus = "Atualizando " + tipo;
    
    // 3 piscadas antes de começar
    for (int i = 0; i < 3; i++) {
      digitalWrite(led, LOW);
      delay(200);
      digitalWrite(led, HIGH);
      delay(200);
    }
  });
  
  ArduinoOTA.onEnd([]() {
    Serial.println("\n✅ Atualização OTA concluída!");
    otaAtivo = false;
    otaStatus = "Concluída - Reiniciando...";
    
    // 3 piscadas no final
    for (int i = 0; i < 3; i++) {
      digitalWrite(led, LOW);
      delay(300);
      digitalWrite(led, HIGH);
      delay(300);
    }
    
    // Garante LED apagado antes de reiniciar
    digitalWrite(led, HIGH);
    delay(1000);
    ESP.restart();
  });
  
  ArduinoOTA.onProgress([](unsigned int progresso, unsigned int total) {
    int percentual = (progresso / (total / 100));
    Serial.printf("📊 Progresso: %u%%\r", percentual);
    otaStatus = "Progresso: " + String(percentual) + "%";
    // LED pisca durante o progresso
    digitalWrite(led, (millis() % 200) < 100 ? LOW : HIGH);
  });
  
  ArduinoOTA.onError([](ota_error_t erro) {
    Serial.printf("❌ Erro OTA[%u]: ", erro);
    if (erro == OTA_AUTH_ERROR) Serial.println("Falha na autenticação");
    else if (erro == OTA_BEGIN_ERROR) Serial.println("Falha ao iniciar");
    else if (erro == OTA_CONNECT_ERROR) Serial.println("Falha na conexão");
    else if (erro == OTA_RECEIVE_ERROR) Serial.println("Falha no recebimento");
    else if (erro == OTA_END_ERROR) Serial.println("Falha ao finalizar");
    otaAtivo = false;
    otaStatus = "Erro na atualização";
  });
  
  ArduinoOTA.begin();
  otaAtivo = true;
  otaStatus = "Pronto para atualização";
  Serial.println("🚀 OTA iniciado - Pronto para atualizações via rede!");
  Serial.println("🔐 Senha OTA: SyraTeam");
  Serial.println("� Hostname: syra.local (porta 8266)");
}

void salvarModo(byte modo) {
  File f = LittleFS.open("/modo.txt", "w");
  if (f) {
    f.print(modo);
    f.close();
    Serial.println("💾 Modo salvo: " + String(modo == 1 ? "AP" : "Cliente"));
  } else {
    Serial.println("❌ Erro ao salvar modo!");
  }
}

byte lerModo() {
  File f = LittleFS.open("/modo.txt", "r");
  if (!f) {
    Serial.println("📄 Nenhum modo salvo, usando padrão (Cliente)");
    return 0; // Padrão: modo cliente
  }
  String conteudo = f.readString();
  conteudo.trim();
  f.close();
  byte modo = conteudo.toInt();
  Serial.println("📂 Modo carregado: " + String(modo == 1 ? "AP" : "Cliente"));
  return modo;
}

void configurarWiFiManager() {
  // Configura callbacks do WiFiManager
  wifiManager.setAPCallback([](WiFiManager *wm) {
    Serial.println(F("🔧 Entrando no modo de configuração WiFi"));
    Serial.println(F("📡 Rede: ") + String(config_ap_ssid));
    Serial.println(F("🔐 Senha: ") + String(config_ap_password));
    Serial.println(F("🌐 Acesse: http://192.168.4.1/"));
    wifiConfigMode = true;
    wifiStatus = "Configuração";
  });
  
  wifiManager.setSaveConfigCallback([]() {
    Serial.println(F("✅ Configuração WiFi salva!"));
    wifiStatus = "Configurado";
  });
  
  // Configura timeout para o portal de configuração (3 minutos)
  wifiManager.setConfigPortalTimeout(180);
  
  // Define informações customizadas
  wifiManager.setHostname("syra");
  wifiManager.setTitle("SyraHome - Configuração WiFi");
  
  Serial.println(F("⚙️ WiFiManager configurado"));
}

void conectarWiFiManager() {
  Serial.println(F("🔍 Tentando conectar com WiFiManager..."));
  
  // Define LED piscando durante configuração
  digitalWrite(led, LOW);
  delay(100);
  digitalWrite(led, HIGH);
  
  // Tenta conectar automaticamente ou abre portal de configuração
  if (!wifiManager.autoConnect(config_ap_ssid, config_ap_password)) {
    Serial.println(F("❌ Falha na configuração WiFi - timeout atingido"));
    Serial.println(F("🔄 Alternando para modo Access Point..."));
    wifiStatus = "Falha - mudando para AP";
    
    // Se falhar, muda para modo AP
    delay(1000);
    criarRedeAP();
    modoAP = true;
    modoManual = true;
    salvarModo(1);
  } else {
    Serial.println(F("✅ Conectado via WiFiManager!"));
    currentSSID = WiFi.SSID();
    wifiStatus = "Conectado";
    wifiConfigMode = false;
    tentativaInicio = millis();
  }
}
