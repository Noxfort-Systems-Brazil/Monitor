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

## 📖 Executive Summary

**Noxfort Monitor™** is an industrial event-driven observability platform developed in Go to monitor distributed systems, autonomous agents (such as **Synapse** and **Carina**), and IoT hardware.

It features a dual-engine database persistence layer (**PostgreSQL** for enterprise network operations and embedded **SQLite** with live hot-reloading), concurrent telemetry ingestion via **MQTT** and **HTTP REST**, silent failure detection through the **Watchdog Engine**, multi-channel alert dispatching with Role-Based Access Control (**RBAC**), secure reverse tunneling via **Ngrok** for edge nodes on remote networks, and a native desktop interface powered by **Wails v2** with **Headless** mode support.

---

## ⚡ Key Features

### 1. Dual Telemetry Ingestion (MQTT + HTTP REST)
- **Asynchronous MQTT Ingestion**: Parallel processing of JSON payloads over `tcp://127.0.0.1:1883` without I/O blocking.
- **Direct REST API**: `POST /api/telemetry` endpoint for sensors, field nodes, and cURL scripts without requiring a native MQTT client.
- **Intelligent Noise Filter**: Keep-alive messages ("*heartbeat*", "*system ok*") update presence timestamps without burdening the database.

### 2. Dual-Engine Persistence (PostgreSQL & SQLite)
- **Live Hot-Reload**: The [`DBManager`](docs/DATABASE.md) allows runtime switching between SQLite and PostgreSQL via the `/server` screen without restarting the process or dropping active connections.
- **Automatic Migration**: Full, consistent synchronization of devices, configurations, users, and contacts between database engines.
- **Pure-Go SQLite**: Built with `modernc.org/sqlite`, eliminating CGO compiler dependencies.

### 3. Remote Access & WAN Ingestion (Ngrok Tunnel)
- **Embedded Secure Tunnel**: Traverses industrial firewalls and CGNAT via the [`internal/tunnel`](docs/REMOTE_ACCESS.md) subsystem, exposing a secure public URL with static domain support at boot.
- **UI Integration**: The `/devices` view automatically adapts suggested ingestion commands with the public tunnel endpoint ready to copy.

### 4. Watchdog Engine (Silent Failure Detection)
- **Active Polling**: Continuously monitors equipment `LastSeen` timestamps. If a system goes silent for more than 5 minutes, it synthesizes a `CRITICAL` `System OFFLINE` incident and automatically resolves it when the signal returns.

### 5. Intelligent Routing & Audit Trail
- **Role-Based Dispatch (RBAC)**: `HARDWARE` alerts are routed to Technicians; `SOFTWARE` alerts are routed to Programmers; Administrators receive complete visibility.
- **Concurrent Multi-Channel**: Goroutine-based delivery via Email (SMTP) and Telegram Bot (MarkdownV2).
- **Triple Audit Trail**: Immutable tracking of security access and logins (`SecurityAuditLog`), alert delivery compliance (`AlertDispatchLog`), and downtime history (`DeviceStateTransition`).

### 6. Wails v2 Desktop Interface & Headless Mode
- **Native Desktop**: Built with Wails v2 and native Linux WebKitGTK, supporting single-instance enforcement (Single-Instance IPC) and minimize-to-system-tray (Systray).
- **Server Mode (Headless)**: Runs as a background daemon using the `--headless` flag for screenless servers and containerized deployments.
- **Debian Packaging**: Built-in `.deb` package generation script for Ubuntu/Debian (`make deb`).

---

## 📚 Technical Documentation Hub

Explore the modular, interconnected technical documentation:

* 🧭 **[Documentation Hub (docs/INDEX.md)](docs/INDEX.md)**: Master table of contents and knowledge graph.
* 🏗️ **[System Architecture (ARCHITECTURE.md)](ARCHITECTURE.md)**: Concurrency, SOLID layers, dependency injection, and data flow.
* 📡 **[API & Protocol Reference](docs/API_REFERENCE.md)**: Formal specification of HTTP REST endpoints and MQTT topics.
* 🗄️ **[Database & Dual-Engine Persistence](docs/DATABASE.md)**: PostgreSQL, SQLite, DBManager, and data migration.
* 🔐 **[Security, Authentication & RBAC](docs/SECURITY.md)**: Sessions, cookies, salted hashing, superuser bootstrapping, and role enforcement.
* 🌐 **[Remote Access & Ngrok Tunnel](docs/REMOTE_ACCESS.md)**: WAN telemetry ingestion for edge nodes on external networks.
* 🖥️ **[Desktop Application & Operations](docs/DESKTOP_APP.md)**: Wails v2, WebKitGTK, single-instance lock, and `.deb` packaging.
* 🔍 **[Audit Trail](docs/AUDIT_TRAIL.md)**: Regulatory compliance, alert delivery SLA, and downtime tracking.
* 🚀 **[Production Deployment Guide](docs/DEPLOYMENT.md)**: Headless systemd service, NGINX reverse proxy with SSL, and enterprise setup.
* 👨‍💻 **[Developer Guide](docs/DEVELOPER_GUIDES.md)**: Local environment setup, Go conventions, and extensibility.
* 🧪 **[Testing & Quality Assurance](docs/TESTING.md)**: Unit testing with mocks, cURL, and mosquitto_pub.
* 🔬 **[Research Notes & Technical Decisions](docs/RESEARCH_NOTES.md)**: Architectural decisions and future roadmap.

---

## ⚙️ Quickstart Guide

### 1. Prerequisites
Ensure you have [Go 1.22+](https://go.dev/dl/) and WebKitGTK libraries installed:
```bash
sudo apt-get update && sudo apt-get install -y \
  libgtk-3-dev libwebkit2gtk-4.1-dev libappindicator3-dev mosquitto
```

### 2. Start the MQTT Broker
```bash
make broker-start
```

### 3. Run in Development Mode (Desktop GUI)
```bash
make build
make run
```

### 4. Run in Server Mode (Headless Daemon)
Ideal for servers without a graphical interface:
```bash
make run-headless
# Or run the compiled binary directly:
./bin/noxfort-monitor --headless
```

### 5. Generate Debian Installer (`.deb`)
```bash
make deb
sudo dpkg -i build_deb/noxfort-monitor_2.0.1_amd64.deb
```

---

## 🔐 Authentication & Default Credentials

* **Web Dashboard**: Accessible at `http://localhost:8080`.
* **Testing / Evaluation Environment**: If no `.env` file is present, the default first-access credentials are:
  * **Username**: `admin`
  * **Password**: `admin`
* **Production Environment**: Copy `.env.example` to `.env` and configure strong passwords prior to deployment:
  ```bash
  cp .env.example .env
  # Edit MONITOR_ADMIN_USER and MONITOR_ADMIN_PASSWORD
  ```
* **Data Storage**: In local SQLite mode, the database file resides at `~/Documentos/Monitor/monitor_logs.db`. In PostgreSQL mode, data resides on the configured database server.

---

## 📜 License & Copyright

This software is licensed under the **GNU Affero General Public License (AGPL) v3.0**.  
Copyright © 2026 Gabriel Moraes - Noxfort Systems. All rights reserved.
