[📚 Central de Documentação](docs/INDEX.md) > **Arquitetura Detalhada do Sistema**

---

# 🏗️ Arquitetura Detalhada do Sistema: Noxfort Monitor™

Este documento fornece uma visão arquitetural aprofundada do **Noxfort Monitor™ v2.0**. Foi elaborado para engenheiros de software, arquitetos de sistemas e mantenedores que necessitam compreender a mecânica interna, padrões **SOLID**, modelos de concorrência e o fluxo de dados do sistema.

---

## 1. Filosofia Arquitetural Macro

O Noxfort Monitor adota uma **Arquitetura Orientada a Eventos (EDA)** estrita, combinada com os princípios **SOLID** e **Injeção de Dependências (DI)** iniciada na raiz de composição em [`cmd/server/main.go`](cmd/server/main.go). O sistema elimina estado global compartilhado e pacotes com inicializações ocultas.

### As Camadas do Sistema:
1. **Camada de Transporte (`internal/transport`)**: Terminação de protocolos de rede (Broker MQTT via Paho e Servidor HTTP REST com AuthMiddleware).
2. **Camada de Lógica do Monitor (`internal/monitor`)**: O "cérebro" reativo (State Manager, Watchdog Engine, Alert Service e Channel Tester).
3. **Camada de Segurança (`internal/security`)**: Gerenciamento de sessões, hash criptográfico de senhas, validação de tokens e controle de papéis (RBAC).
4. **Camada de Acesso Remoto (`internal/tunnel`)**: Túnel reverso seguro via Ngrok para ingestão de nós externos عبر WAN.
5. **Camada de Domínio (`internal/domain`)**: Estruturas de dados universais e contratos de interfaces desacopladas.
6. **Camada de Persistência (`internal/storage`)**: Gerenciador dinâmico dual-engine ([`DBManager`](docs/DATABASE.md)), implementações PostgreSQL e SQLite, adaptador de dialetos e migrador de dados.
7. **Camada de Interface Desktop (`internal/desktop`, `internal/tray`)**: Runtime nativo em Wails v2 com WebKitGTK e controle de instância única.

```mermaid
graph TD
    subgraph "Mundo Externo & Nós de Borda"
        LocalDevice[Dispositivo Local / LAN]
        RemoteDevice[Agente Remoto / WAN (Carina, Synapse)]
        Operator[Navegador / Operador Humano]
    end

    subgraph "Camada de Transporte & Rede"
        MQTT[Broker MQTT :1883]
        Ngrok[Túnel Ngrok / WAN HTTPS]
        HTTP[Servidor HTTP :8080]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Camada de Lógica & Segurança"
        StateManager[State Manager]
        Watchdog[Watchdog Engine]
        Alerts[Alert Routing Service]
        SecManager[Security Manager]
        TunnelMgr[Tunnel Manager]
    end

    subgraph "Persistência Dual-Engine (internal/storage)"
        DBMgr[DBManager Central]
        AuditRepo[AuditRepository]
        PG[(PostgreSQL Industrial)]
        SQLite[(SQLite Pure-Go)]
    end

    subgraph "Canais de Notificação Externa"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[Servidor SMTP (Email)]
    end

    LocalDevice -- "MQTT Publish" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Decodifica Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persiste Incidente" --> DBMgr
    StateManager -- "2. Dispara Alerta" --> Alerts
    Watchdog -- "Verifica Heartbeats" --> DBMgr
    Watchdog -- "Sintetiza Queda/Recuperação" --> Alerts
    Watchdog -- "Registra Transição" --> AuditRepo

    Alerts -- "Goroutine Concorrente" --> Telegram
    Alerts -- "Goroutine Concorrente" --> Email
    Alerts -- "Registra SLA de Envio" --> AuditRepo

    SecManager -- "Audita Logins" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Subsistemas Centrais em Detalhes

### 2.1 O Gerenciador de Estados (`internal/monitor/state.go`)
O `StateManager` é o ponto focal de roteamento de eventos. Ele recebe cargas úteis decodificadas da camada de transporte (MQTT ou HTTP REST) e aplica o fluxo "Filtrar e Agir":
* **Filtro de Heartbeat**: Toda mensagem recebida atualiza imediatamente o campo `last_seen` do dispositivo de origem através de `UpdateLastSeen`. O detector [`KeywordHeartbeatDetector`](internal/monitor/state.go) avalia a mensagem: se for de nível `INFO` e contiver palavras-chave de keep-alive ("*system ok*", "*heartbeat*", "*online*"), o processamento é finalizado ali, evitando consumo inútil de armazenamento e fadiga de alertas.
* **Processamento de Incidentes**: Se a mensagem for um incidente real, o `StateManager` persiste o evento no repositório de telemetria e o encaminha para o `AlertService`.

### 2.2 O Motor Watchdog (`internal/monitor/engine.go`)
Enquanto o State Manager lida com eventos ativos, o `Engine` é responsável por detectar **falhas silenciosas**:
* **Concorrência**: Executa em uma goroutine dedicada, acionada por um `time.Ticker` com intervalo padrão de 30 segundos.
* **Avaliação de Presença**: Compara a hora atual com o `LastSeen` de cada equipamento. Se um sistema habilitado ficar sem reportar por mais de **5 minutos**, a estrutura [`SystemStatusTracker`](internal/monitor/tracker.go) detecta a transição e sintetiza um incidente `CRITICAL` `System OFFLINE`.
* **Detecção de Recuperação**: Quando um sistema que estava offline volta a enviar sinais, o Engine sintetiza um evento `INFO` de recuperação e registra a duração total do tempo de inatividade (*downtime*) no repositório de auditoria.

### 2.3 Roteamento Inteligente de Alertas (`internal/monitor/alerts.go`)
O `AlertService` desacopla a regra de notificação do envio físico através da interface [`NotificationChannel`](internal/monitor/channel.go):
* **Categorização por Papel (RBAC)**:
  * **Administradores**: Recebem todos os incidentes globais.
  * **Técnicos**: Recebem apenas alertas da categoria `HARDWARE`.
  * **Programadores**: Recebem apenas alertas da categoria `SOFTWARE`.
* **Filtro de Severidade**: Contatos podem configurar seus perfis para receber exclusivamente alertas `CRITICAL`, suprimindo avisos `WARNING`.
* **Despacho Assíncrono**: Cada notificação para cada contato é enviada em sua própria goroutine, garantindo que servidores SMTP lentos não causem atrasos no broker MQTT ou na API do Telegram.
* **Auditoria de Entrega**: Toda tentativa gera um registro [`AlertDispatchLog`](docs/AUDIT_TRAIL.md) com status `SENT` ou `FAILED` e motivo da recusa.

### 2.4 Subsistema de Segurança & Sessões (`internal/security`)
* **Gerenciamento de Identidade**: [`SecurityManager`](internal/security/security_manager.go) centraliza autenticação, RBAC e auditoria.
* **Tokens Criptográficos**: O [`SessionManager`](internal/security/session.go) emite tokens seguros em memória com renovação e expiração.
* **Isolamento de Senhas**: Hashes de senha contêm salt exclusivo e são omitidos em qualquer serialização JSON.

### 2.5 Acesso Remoto & Ingestão WAN (`internal/tunnel`)
* O pacote [`internal/tunnel`](docs/REMOTE_ACCESS.md) encapsula o driver Ngrok, iniciando túneis com domínios estáticos e mantendo o status em memória para distribuição aos clientes de campo.

---

## 3. Modelo de Domínio e Entidades Centrais (`internal/domain`)

O pacote `internal/domain` não possui nenhuma dependência externa, constituindo o núcleo puro da aplicação:

* **`IncomingEvent`**: A estrutura universal de telemetria contendo `Category`, `Origin`, `Level`, `Message` e `OccurredAt`.
* **`Device`**: Representa um nó monitorado, com seu nome amigável, identificador e carimbo de data/hora `LastSeen`.
* **`Contact`**: Destinatários de incidentes com seus papéis, canais (Email, Telegram Chat ID) e filtros de severidade.
* **`User`**: Operadores do sistema com suas credenciais e papéis (`RoleAdmin`, `RoleOperator`).
* **`SecurityAuditLog`**, **`AlertDispatchLog`**, **`DeviceStateTransition`**: Modelos de auditoria e conformidade.
* **`DatabaseConfig`** e **`DatabaseStatus`**: Parâmetros e estado de saúde da camada de dados.

---

## 4. Camada de Persistência & Dual-Engine (`internal/storage`)

Consulte o guia dedicado [Banco de Dados & Persistência](docs/DATABASE.md).

* **Dual-Engine Nativo**:
  * **SQLite**: `modernc.org/sqlite` para modo embarcado sem compilador C (CGO-free).
  * **PostgreSQL**: `github.com/lib/pq` para servidores industriais com isolamento de schema (`schema_monitor`).
* **DBManager Central**: Permite alternância a quente de banco via `ReloadableRepository.SetDB()` sem reiniciar o processo.
* **Migrador Automático**: Sincronização estruturada de dados entre drivers via `MigrateData()`.
* **Query Adapter**: Adaptação em tempo de execução de placeholders `?` para `$1, $2` e resolução de cláusulas de conflito.

---

## 5. Modelo de Threads, Concorrência & Sistema Operacional

```mermaid
graph TD
    Main[Main OS Thread / Goroutine Principal]
    
    Main -->|Modo GUI Padrão| WailsEventLoop[Wails v2 Desktop Event Loop]
    WailsEventLoop --> Systray[Systray GTK Callbacks]
    WailsEventLoop --> WebKit[Janela WebKitGTK]
    
    Main -->|Modo --headless| SigChan[Signal Notify Loop (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[Servidor HTTP ListenAndServe]
    Main -.->|go func| MQTTListener[Paho MQTT Packet Loop]
    Main -.->|go func| WatchdogEngine[Ticker Loop - 30s Interval]
    Main -.->|go func| AlertWorkers[Workers Concorrentes de Email/Telegram]
```

* **Goroutine Principal**: 
  * Em modo desktop, executa `desktopApp.Run()` (Wails v2), obrigatório pois os toolkits gráficos do Linux (GTK/WebKit) exigem execução na thread principal do sistema operacional.
  * Em modo `--headless`, bloqueia em um canal de sinais do sistema operacional (`syscall.SIGTERM`, `os.Interrupt`).
* **Servidor Web**: Opera em goroutine independente com timeouts de leitura/escrita de 15 segundos.
* **Listener MQTT**: O cliente Paho gerencia o socket TCP em goroutines dedicadas de leitura/escrita.
* **Loop do Engine**: Opera em canal `time.Ticker` desacoplado.
* **Encerramento Gracioso (Graceful Shutdown)**: Coordenado por uma função segura com `sync.Once` que fecha o socket IPC, para o túnel Ngrok, encerra o servidor HTTP, para o Engine, desconecta do broker MQTT e libera o banco de dados.

---

### 🔗 Documentos Relacionados
* 🧭 [Central de Documentação](docs/INDEX.md) — Índice geral do repositório
* 🗄️ [Banco de Dados & Persistência](docs/DATABASE.md) — DBManager, PostgreSQL e SQLite
* 🔐 [Segurança e RBAC](docs/SECURITY.md) — Detalhes do SecurityManager e Middleware
* 🌐 [Acesso Remoto](docs/REMOTE_ACCESS.md) — Arquitetura do túnel Ngrok
* 🖥️ [Aplicação Desktop](docs/DESKTOP_APP.md) — Runtime Wails v2 e modo Headless
* 🔍 [Trilha de Auditoria](docs/AUDIT_TRAIL.md) — Modelo de conformidade
* 📡 [Referência de APIs](docs/API_REFERENCE.md) — Contratos REST e MQTT
