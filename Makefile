# Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
# Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# File: Makefile
# Author: Gabriel Moraes
# Date: 2026-01-13

# Binary output name
BINARY_NAME=bin/noxfort-server
MAIN_PATH=cmd/server/main.go
MOSQUITTO_CONF=mosquitto/config/mosquitto.conf

# Default target (what runs when you just type 'make')
all: build

# 1. Build the executable
build:
	@echo "🔨 Building Noxfort Monitor..."
	@mkdir -p bin
	go build -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build successful! Binary created at $(BINARY_NAME)"

# 2. Run the application directly (Development mode)
#    Make sure to run 'make broker-start' first.
run:
	@echo "🚀 Running Server... (ensure broker is running: make broker-start)"
	go run $(MAIN_PATH)

# 3. Clean up build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f monitor_logs.db
	@echo "✨ Clean complete."

# 4. Run automated tests
test:
	@echo "🧪 Running Tests..."
	go test ./... -v

# 5. Install/Update Dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod tidy
	go mod download

# 6. Cross-compile for Linux
build-linux:
	@echo "🐧 Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux $(MAIN_PATH)
	@echo "✅ Linux binary ready."

# ---- MQTT Broker (Mosquitto - native, no Docker required) ----

# 7. Start the Mosquitto MQTT broker
#    Uses systemctl if available (recommended), otherwise runs directly.
broker-start:
	@echo "🟢 Starting Mosquitto MQTT broker..."
	@mkdir -p mosquitto/data mosquitto/log
	@if systemctl is-active --quiet mosquitto 2>/dev/null; then \
		echo "✅ Mosquitto is already running via systemd."; \
	elif command -v systemctl >/dev/null 2>&1; then \
		sudo systemctl start mosquitto && echo "✅ Broker started via systemd."; \
	else \
		mosquitto -c $(MOSQUITTO_CONF) -d && echo "✅ Broker started in background."; \
	fi

# 8. Stop the Mosquitto MQTT broker
broker-stop:
	@echo "🔴 Stopping Mosquitto MQTT broker..."
	@if command -v systemctl >/dev/null 2>&1; then \
		sudo systemctl stop mosquitto && echo "✅ Broker stopped."; \
	else \
		pkill -x mosquitto && echo "✅ Broker stopped." || echo "⚠️  Mosquitto was not running."; \
	fi

# 9. Check if the Mosquitto broker is running
broker-status:
	@echo "📡 MQTT Broker status:"
	@if systemctl is-active --quiet mosquitto 2>/dev/null; then \
		echo "  ✅ Mosquitto is running (systemd service)"; \
	elif pgrep -x mosquitto > /dev/null; then \
		echo "  ✅ Mosquitto is running (standalone process, PID: $$(pgrep -x mosquitto))"; \
	else \
		echo "  ❌ Mosquitto is NOT running. Start it with: make broker-start"; \
	fi

# 10. Install Mosquitto (if not already installed)
broker-install:
	@echo "📦 Installing Mosquitto..."
	sudo apt-get update -qq && sudo apt-get install -y mosquitto
	@echo "✅ Mosquitto installed."

.PHONY: all build run clean test deps build-linux broker-start broker-stop broker-status broker-install