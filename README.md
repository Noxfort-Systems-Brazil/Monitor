# Noxfort Monitor™ Server

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Version](https://img.shields.io/badge/Version-2.0.0-brightgreen.svg)]()
[![Platform](https://img.shields.io/badge/Platform-Ubuntu_22.04_LTS-E95420?style=flat&logo=ubuntu&logoColor=white)]()

**Noxfort Monitor™** is an open-source industrial telemetry, observability, and incident response orchestration system. Built in Go, it features a highly efficient event-driven architecture designed to monitor devices, trigger alerts, and provide a comprehensive web-based dashboard and system tray integration.

## 🚀 Key Features

*   **Event-Driven Architecture**: Fast and lightweight telemetry ingestion.
*   **MQTT Integration**: Subscribes to device telemetry events via MQTT brokers like Mosquitto.
*   **Built-in Web Interface**: Real-time HTTP web dashboard for monitoring operations.
*   **System Tray Integration**: Native tray icon for Linux desktop environments for easy management and quick exit.
*   **Embedded Database**: Uses a pure Go SQLite driver (`modernc.org/sqlite`), automatically keeping user data securely at `~/Documentos/Monitor/monitor_logs.db`.
*   **Alert Generation**: Evaluates rules against telemetry to orchestrate incident response.

## 🛠️ Technology Stack

*   **Language**: Go 1.22+
*   **Database**: SQLite (Pure Go implementation)
*   **Messaging**: MQTT (Eclipse Paho)
*   **Interface**: HTML/Templates (Web API) & Desktop System Tray (`getlantern/systray`)

## 📂 Project Structure

*   `cmd/server/`: Main application entry point (`main.go`).
*   `internal/`: Core business logic (domains, state managers, storage repositories).
*   `mosquitto/`: Bundled configurations and data directories for the MQTT broker.
*   `web/`: Web interface assets (static files and HTML templates).
*   `configs/`: Application configuration setups.

## ⚙️ Getting Started

### Prerequisites
*   [Go 1.22+](https://go.dev/dl/)
*   [Mosquitto MQTT Broker](https://mosquitto.org/download/) (or use the provided `docker-compose.yml`)

### 1. Setup the MQTT Broker
You need an MQTT broker running locally. The project includes a target to run Mosquitto natively:
```bash
make broker-install # If you don't have Mosquitto installed
make broker-start   # Starts the broker service
```
*Alternatively, you can use docker:*
```bash
docker-compose up -d mosquitto
```

### 2. Build the Application
```bash
make build
```
The binary will be generated at `bin/noxfort-server`.

### 3. Run the Server
```bash
make run
```
*   **Web Dashboard**: [http://localhost:8080](http://localhost:8080)
*   **MQTT Broker**: Connects by default to `tcp://127.0.0.1:1883`

## 🧪 Testing and Development

*   **Run Automated Tests**: 
    ```bash
    make test
    ```
*   **Cross-compile for Release (Linux AMD64)**:
    ```bash
    make build-linux
    ```

## 📜 License

This program is free software: you can redistribute it and/or modify it under the terms of the **GNU Affero General Public License as published by the Free Software Foundation**, either version 3 of the License, or (at your option) any later version.

Copyright © 2026 Gabriel Moraes - Noxfort Systems
