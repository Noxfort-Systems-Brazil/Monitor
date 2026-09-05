<div align="center">
  <h1>🛡️ Noxfort Monitor™ Server v2.0</h1>
  <p><strong>Industrial Telemetry Ingestion, Observability & Incident Response Orchestration</strong></p>

  [![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
  [![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg?style=for-the-badge)](https://www.gnu.org/licenses/agpl-3.0)
  [![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=for-the-badge&logo=ubuntu&logoColor=white)]()
  [![Desktop](https://img.shields.io/badge/GUI-Wails_v2-df0000?style=for-the-badge)]()
  [![Architecture](https://img.shields.io/badge/Architecture-Event--Driven-8A2BE2?style=for-the-badge)]()
</div>

<hr/>

## 📖 Resumo Executivo

O **Noxfort Monitor™** é uma plataforma industrial de observabilidade orientada a eventos (*Event-Driven*), desenvolvida em Go para monitorar sistemas distribuídos, agentes autônomos (como **Synapse** e **Carina**) e hardware IoT. 

Possui motor duplo de banco de dados (**PostgreSQL** para operação corporativa em rede e **SQLite** embarcado com hot-reload a quente), ingestão simultânea via **MQTT** e **HTTP REST**, detecção de falhas silenciosas via **Watchdog Engine**, despacho de alertas multicanal com controle de acesso baseado em papéis (**RBAC**), túnel reverso seguro via **Ngrok** para nós de borda em redes remotas, e interface nativa em **Wails v2** com suporte a modo **Headless**.

---

## ⚡ Principais Funcionalidades

### 1. Ingestão Dual de Telemetria (MQTT + HTTP REST)
- **Ingestão Assíncrona MQTT**: Processamento paralelo de pacotes JSON sobre `tcp://127.0.0.1:1883` sem bloqueio de I/O.
- **API REST Direta**: Endpoint `POST /api/telemetry` para sensores, nós de campo e scripts cURL sem necessidade de client MQTT nativo.
- **Filtro Inteligente de Ruído**: Mensagens de keep-alive ("*heartbeat*", "*system ok*") atualizam o timestamp de presença sem sobrecarregar o banco de dados.

### 2. Motor de Persistência Dual-Engine (PostgreSQL & SQLite)
- **Hot-Reload a Quente**: O [`DBManager`](docs/DATABASE.md) permite alternar entre SQLite e PostgreSQL em tempo de execução via tela `/server` sem reiniciar o processo ou derrubar conexões.
- **Migração Automática**: Sincronização íntegra de dispositivos, configurações, usuários e contatos entre bancos de dados.
- **SQLite Pure-Go**: Utiliza `modernc.org/sqlite`, eliminando dependências de compilador CGO.

### 3. Acesso Remoto & Ingestão WAN (Túnel Ngrok)
- **Túnel Seguro Embutido**: Transpõe firewalls e CGNAT industriais através do subsistema [`internal/tunnel`](docs/REMOTE_ACCESS.md), expondo uma URL pública segura com domínio estático no boot.
- **Integração na UI**: A tela `/devices` adapta os comandos sugeridos com o endpoint público do túnel pronto para cópia.

### 4. Watchdog Engine (Detecção de Falhas Silenciosas)
- **Varredura Ativa**: Monitora continuamente o timestamp `LastSeen` dos equipamentos. Se um sistema silenciar por mais de 5 minutos, sintetiza um incidente `CRITICAL` `System OFFLINE` e auto-resolve quando o sinal retorna.

### 5. Roteamento Inteligente & Trilha de Auditoria
- **Despacho Baseado em Funções (RBAC)**: Alertas `HARDWARE` são roteados para Técnicos; alertas `SOFTWARE` para Programadores; Administradores recebem visibilidade total.
- **Multicanal Concorrente**: Envio em goroutines via Email (SMTP) e Telegram Bot (MarkdownV2).
- **Trilha de Auditoria Tripla**: Rastreamento imutável de acessos e logins (`SecurityAuditLog`), conformidade de entrega de alertas (`AlertDispatchLog`) e histórico de downtime (`DeviceStateTransition`).

### 6. Interface Desktop Wails v2 & Modo Headless
- **Desktop Nativo**: Construído com Wails v2 e WebKitGTK nativo do Linux, suporte a trava de instância única (Single-Instance IPC) e minimização para a barra de tarefas (Systray).
- **Modo Servidor (Headless)**: Executa como daemon em segundo plano com a flag `--headless` para servidores e contêineres sem display gráfico.
- **Pacote Debian**: Script de compilação de pacote `.deb` integrado para Ubuntu/Debian (`make deb`).

---

## 📚 Biblioteca de Documentação Técnica

Navegue pela documentação técnica modular e interconectada:

* 🧭 **[Central de Documentação (docs/INDEX.md)](docs/INDEX.md)**: Mapa mestre de conteúdo e grafo de conhecimento.
* 🏗️ **[Arquitetura do Sistema (ARCHITECTURE.md)](ARCHITECTURE.md)**: Concorrência, camadas SOLID, injeção de dependência e fluxo de dados.
* 📡 **[Referência de APIs & Protocolos](docs/API_REFERENCE.md)**: Especificação formal dos endpoints HTTP REST e tópicos MQTT.
* 🗄️ **[Banco de Dados & Persistência Dual-Engine](docs/DATABASE.md)**: PostgreSQL, SQLite, DBManager e migração de dados.
* 🔐 **[Segurança, Autenticação & RBAC](docs/SECURITY.md)**: Sessões, cookies, hashing com salt, superusuário e controle de papéis.
* 🌐 **[Acesso Remoto & Túnel Ngrok](docs/REMOTE_ACCESS.md)**: Ingestão de telemetria WAN para nós de borda em redes externas.
* 🖥️ **[Aplicação Desktop & Operação](docs/DESKTOP_APP.md)**: Wails v2, WebKitGTK, single-instance lock e instalador `.deb`.
* 🔍 **[Trilha de Auditoria](docs/AUDIT_TRAIL.md)**: Conformidade regulatória, SLA de entrega de alertas e histórico de paradas.
* 🚀 **[Guia de Implantação em Produção](docs/DEPLOYMENT.md)**: Serviço systemd headless, proxy NGINX com SSL e setup corporativo.
* 👨‍💻 **[Guia do Desenvolvedor](docs/DEVELOPER_GUIDES.md)**: Configuração do ambiente local, convenções Go e extensibilidade.
* 🧪 **[Testes & Garantia de Qualidade](docs/TESTING.md)**: Testes unitários com mocks, cURL e mosquitto_pub.
* 🔬 **[Notas de Pesquisa & Decisões Técnicas](docs/RESEARCH_NOTES.md)**: Decisões arquiteturais e roadmap futuro.

---

## ⚙️ Guia de Início Rápido

### 1. Pré-Requisitos
Certifique-se de possuir [Go 1.22+](https://go.dev/dl/) e as bibliotecas do WebKitGTK instaladas:
```bash
sudo apt-get update && sudo apt-get install -y \
  libgtk-3-dev libwebkit2gtk-4.1-dev libappindicator3-dev mosquitto
```

### 2. Iniciar o Broker MQTT
```bash
make broker-start
```

### 3. Execução em Desenvolvimento (Desktop GUI)
```bash
make build
make run
```

### 4. Execução em Modo Servidor (Headless Daemon)
Ideal para servidores sem interface gráfica:
```bash
make run-headless
# Ou executando o binário compilado:
./bin/noxfort-monitor --headless
```

### 5. Gerar Instalador Debian (`.deb`)
```bash
make deb
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
```

---

## 🔐 Autenticação & Credenciais Padrão

* **Painel Web**: Acessível em `http://localhost:8080`.
* **Ambiente de Testes / Avaliação**: Se nenhum arquivo `.env` for configurado, as credenciais padrão de primeiro acesso são:
  * **Usuário**: `admin`
  * **Senha**: `admin`
* **Ambiente de Produção**: Copie `.env.example` para `.env` e configure senhas fortes antes da implantação:
  ```bash
  cp .env.example .env
  # Edite MONITOR_ADMIN_USER e MONITOR_ADMIN_PASSWORD
  ```
* **Armazenamento de Dados**: No modo SQLite local, o arquivo de banco reside em `~/Documentos/Monitor/monitor_logs.db`. No modo PostgreSQL, os dados residem no servidor de banco de dados configurado.

---

## 📜 Licença & Direitos Autorais

Este software é licenciado sob a **GNU Affero General Public License (AGPL) v3.0**.  
Copyright © 2026 Gabriel Moraes - Noxfort Systems. Todos os direitos reservados.
