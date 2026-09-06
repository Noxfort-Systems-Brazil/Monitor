[📚 Documentation Hub](docs/INDEX.md) > **Detailed System Architecture**

---

# 🏗️ Detailed System Architecture: Noxfort Monitor™

This document provides an in-depth architectural overview of **Noxfort Monitor™ v2.0**. It is intended for software engineers, systems architects, and maintainers who need to understand the internal mechanics, **SOLID** patterns, concurrency models, and data flows of the system.

---

## 1. Macro Architectural Philosophy

Noxfort Monitor adheres to a strict **Event-Driven Architecture (EDA)** combined with **SOLID** principles and **Dependency Injection (DI)** assembled at the composition root in [`cmd/server/main.go`](cmd/server/main.go). The system eliminates shared global mutable state and packages with hidden initialization routines.

### System Layers:
1. **Transport Layer (`internal/transport`)**: Network protocol termination (MQTT Broker via Paho and HTTP REST Server with AuthMiddleware).
2. **Monitor Logic Layer (`internal/monitor`)**: The reactive "brain" (State Manager, Watchdog Engine, Alert Service, and Channel Tester).
3. **Security Layer (`internal/security`)**: Session management, cryptographic password hashing, token validation, and Role-Based Access Control (RBAC).
4. **Remote Access Layer (`internal/tunnel`)**: Secure reverse tunneling via Ngrok for WAN-based edge node ingestion.
5. **Domain Layer (`internal/domain`)**: Universal data structures and decoupled interface contracts.
6. **Persistence Layer (`internal/storage`)**: Dynamic dual-engine manager ([`DBManager`](docs/DATABASE.md)), PostgreSQL and SQLite implementations, dialect adapter, and data migrator.
7. **Desktop Interface Layer (`internal/desktop`, `internal/tray`)**: Native runtime in Wails v2 with WebKitGTK and single-instance enforcement.

```mermaid
graph TD
    subgraph "External World & Edge Nodes"
        LocalDevice[Local Device / LAN]
        RemoteDevice[Remote Agent / WAN (Carina, Synapse)]
        Operator[Browser / Human Operator]
    end

    subgraph "Transport & Network Layer"
        MQTT[MQTT Broker :1883]
        Ngrok[Ngrok Tunnel / WAN HTTPS]
        HTTP[HTTP Server :8080]
        AuthMW[AuthMiddleware - RBAC]
    end

    subgraph "Logic & Security Layer"
        StateManager[State Manager]
        Watchdog[Watchdog Engine]
        Alerts[Alert Routing Service]
        SecManager[Security Manager]
        TunnelMgr[Tunnel Manager]
    end

    subgraph "Dual-Engine Persistence (internal/storage)"
        DBMgr[Central DBManager]
        AuditRepo[AuditRepository]
        PG[(Industrial PostgreSQL)]
        SQLite[(Pure-Go SQLite)]
    end

    subgraph "External Notification Channels"
        Telegram[Telegram Bot API (MarkdownV2)]
        Email[SMTP Server (Email)]
    end

    LocalDevice -- "MQTT Publish" --> MQTT
    RemoteDevice -- "HTTPS POST /api/telemetry" --> Ngrok
    Ngrok --> HTTP
    Operator -- "HTTP GET / POST" --> HTTP

    MQTT -- "Decodes Payload" --> StateManager
    HTTP --> AuthMW
    AuthMW --> StateManager
    AuthMW --> SecManager

    StateManager -- "1. Persists Incident" --> DBMgr
    StateManager -- "2. Dispatches Alert" --> Alerts
    Watchdog -- "Checks Heartbeats" --> DBMgr
    Watchdog -- "Synthesizes Outage/Recovery" --> Alerts
    Watchdog -- "Records Transition" --> AuditRepo

    Alerts -- "Concurrent Goroutine" --> Telegram
    Alerts -- "Concurrent Goroutine" --> Email
    Alerts -- "Records Dispatch SLA" --> AuditRepo

    SecManager -- "Audits Logins" --> AuditRepo
    DBMgr --> PG
    DBMgr --> SQLite
```

---

## 2. Core Subsystems in Detail

### 2.1 The State Manager (`internal/monitor/state.go`)
The `StateManager` is the central event routing hub. It receives decoded payloads from the transport layer (MQTT or HTTP REST) and applies a "Filter and Act" pipeline:
* **Heartbeat Filtering**: Every incoming message immediately updates the originating device's `last_seen` timestamp via `UpdateLastSeen`. The [`KeywordHeartbeatDetector`](internal/monitor/state.go) evaluates the message: if it is at the `INFO` level and contains keep-alive keywords ("*system ok*", "*heartbeat*", "*online*"), processing completes immediately, preventing redundant database writes and alert fatigue.
* **Incident Processing**: If the message represents a genuine operational incident, the `StateManager` persists the event to the telemetry repository and forwards it to the `AlertService`.

### 2.2 The Watchdog Engine (`internal/monitor/engine.go`)
While the State Manager processes active incoming events, the `Engine` is responsible for detecting **silent failures**:
* **Concurrency**: Runs in a dedicated goroutine driven by a `time.Ticker` with a default 30-second interval.
* **Presence Evaluation**: Compares the current timestamp against the `LastSeen` timestamp of each monitored device. If an enabled device fails to report for more than **5 minutes**, the [`SystemStatusTracker`](internal/monitor/tracker.go) flags the transition and synthesizes a `CRITICAL` `System OFFLINE` incident.
* **Recovery Detection**: When a previously offline device resumes transmitting, the Engine synthesizes an `INFO` recovery event and records the total downtime duration in the audit repository.

### 2.3 Intelligent Alert Routing (`internal/monitor/alerts.go`)
The `AlertService` decouples notification business logic from physical transmission through the [`NotificationChannel`](internal/monitor/channel.go) interface:
* **Role-Based Categorization (RBAC)**:
  * **Administrators**: Receive all global incidents.
  * **Technicians**: Receive only alerts in the `HARDWARE` category.
  * **Programmers**: Receive only alerts in the `SOFTWARE` category.
* **Severity Filtering**: Contacts can configure their profiles to exclusively receive `CRITICAL` alerts, suppressing `WARNING` notifications.
* **Asynchronous Dispatch**: Each notification to each recipient is dispatched in its own goroutine, ensuring that slow SMTP servers never block the MQTT broker or the Telegram API.
* **Delivery Auditing**: Every transmission attempt generates an [`AlertDispatchLog`](docs/AUDIT_TRAIL.md) entry with status `SENT` or `FAILED` along with the failure reason if applicable.

### 2.4 Security & Session Subsystem (`internal/security`)
* **Identity Management**: [`SecurityManager`](internal/security/security_manager.go) centralizes authentication, RBAC, and security auditing.
* **Cryptographic Tokens**: The [`SessionManager`](internal/security/session.go) issues secure in-memory tokens with sliding renewal and expiration.
* **Password Isolation**: Passwords are hashed with unique cryptographic salts and excluded from JSON serialization.

### 2.5 Remote Access & WAN Ingestion (`internal/tunnel`)
* The [`internal/tunnel`](docs/REMOTE_ACCESS.md) package wraps the Ngrok driver, managing secure tunnels with static domains and exposing tunnel status in memory for edge clients.

---

## 3. Domain Model and Core Entities (`internal/domain`)

The `internal/domain` package has zero external dependencies, serving as the pure core of the application:

* **`IncomingEvent`**: Universal telemetry payload containing `Category`, `Origin`, `Level`, `Message`, and `OccurredAt`.
* **`Device`**: Monitored node with its human-readable name, unique identifier, and `LastSeen` timestamp.
* **`Contact`**: Incident recipients with roles, notification channels (Email, Telegram Chat ID), and severity filters.
* **`User`**: System operators with credentials and assigned roles (`RoleAdmin`, `RoleOperator`).
* **`SecurityAuditLog`**, **`AlertDispatchLog`**, **`DeviceStateTransition`**: Compliance and audit trail models.
* **`DatabaseConfig`** and **`DatabaseStatus`**: Parameters and health state of the persistence layer.

---

## 4. Persistence Layer & Dual-Engine (`internal/storage`)

See the dedicated guide [Database & Persistence](docs/DATABASE.md).

* **Native Dual-Engine**:
  * **SQLite**: `modernc.org/sqlite` for embedded deployments without a C compiler (CGO-free).
  * **PostgreSQL**: `github.com/lib/pq` for industrial servers with schema isolation (`schema_monitor`).
* **Central DBManager**: Enables hot database switching via `ReloadableRepository.SetDB()` without restarting the process.
* **Automatic Migrator**: Structured data synchronization across engines via `MigrateData()`.
* **Query Adapter**: Runtime adaptation of SQL placeholders (`?` to `$1, $2`) and conflict clause resolution.

---

## 5. Threading Model, Concurrency & Operating System

```mermaid
graph TD
    Main[Main OS Thread / Primary Goroutine]
    
    Main -->|Standard GUI Mode| WailsEventLoop[Wails v2 Desktop Event Loop]
    WailsEventLoop --> Systray[Systray GTK Callbacks]
    WailsEventLoop --> WebKit[WebKitGTK Window]
    
    Main -->|--headless Mode| SigChan[Signal Notify Loop (SIGINT/SIGTERM)]

    Main -.->|go func| HTTPServer[HTTP Server ListenAndServe]
    Main -.->|go func| MQTTListener[Paho MQTT Packet Loop]
    Main -.->|go func| WatchdogEngine[Ticker Loop - 30s Interval]
    Main -.->|go func| AlertWorkers[Concurrent Email/Telegram Workers]
```

* **Primary Goroutine**: 
  * In desktop mode, executes `desktopApp.Run()` (Wails v2), required because Linux graphical toolkits (GTK/WebKit) must run on the primary OS thread.
  * In `--headless` mode, blocks on an OS signal notification channel (`syscall.SIGTERM`, `os.Interrupt`).
* **Web Server**: Runs in an independent goroutine with 15-second read/write timeouts.
* **MQTT Listener**: The Paho client manages the TCP socket across dedicated read/write goroutines.
* **Watchdog Loop**: Operates on a decoupled `time.Ticker` channel.
* **Graceful Shutdown**: Coordinated by a thread-safe `sync.Once` routine that closes the IPC socket, stops the Ngrok tunnel, shuts down the HTTP server, stops the Watchdog Engine, disconnects from the MQTT broker, and releases database connections.

---

### 🔗 Related Documentation
* 🧭 [Documentation Hub](docs/INDEX.md) — Master documentation index
* 🗄️ [Database & Persistence](docs/DATABASE.md) — DBManager, PostgreSQL, and SQLite
* 🔐 [Security & RBAC](docs/SECURITY.md) — SecurityManager and Middleware details
* 🌐 [Remote Access](docs/REMOTE_ACCESS.md) — Ngrok reverse tunnel architecture
* 🖥️ [Desktop Application](docs/DESKTOP_APP.md) — Wails v2 runtime and Headless mode
* 🔍 [Audit Trail](docs/AUDIT_TRAIL.md) — Compliance and audit tracking model
* 📡 [API Reference](docs/API_REFERENCE.md) — REST and MQTT contracts
