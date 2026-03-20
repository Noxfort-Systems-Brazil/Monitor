// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/transport/mqtt/client.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps the Paho MQTT client to provide specific Noxfort functionality.
// It bridges the gap between raw MQTT messages and the Domain Logic.
type Client struct {
	internalClient mqtt.Client
	stateManager   *monitor.StateManager
	topicPattern   string
}

// NewClient creates a configured MQTT client instance.
// It subscribes to a wildcard topic to catch all system events.
func NewClient(brokerURL string, sm *monitor.StateManager) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID("noxfort-monitor-server")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("[MQTT] Connection established.")
	})

	c := mqtt.NewClient(opts)

	return &Client{
		internalClient: c,
		stateManager:   sm,
		// Wildcard subscription: Catch EVERYTHING under noxfort/telemetry/
		// We rely on the JSON "origin" field to identify the source.
		topicPattern: "noxfort/telemetry/#",
	}
}

// Connect establishes the connection and subscribes to the telemetry topic.
func (c *Client) Connect() error {
	if token := c.internalClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %w", token.Error())
	}

	// Subscribe using the wildcard pattern
	if token := c.internalClient.Subscribe(c.topicPattern, 1, c.handleMessage); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.topicPattern, token.Error())
	}

	log.Printf("[MQTT] Subscribed to %s (Listening for JSON events)", c.topicPattern)
	return nil
}

// handleMessage is the pipeline entry point: Raw JSON -> IncomingEvent -> State Manager.
func (c *Client) handleMessage(client mqtt.Client, msg mqtt.Message) {
	payload := msg.Payload()

	// 1. Decode the JSON packet directly into the Domain Struct
	// This implements the "Universal JSON" logic.
	var event domain.IncomingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("[MQTT] Decode Error on topic %s: %v", msg.Topic(), err)
		return
	}

	// 2. Validate Origin
	// The "Origin" field (e.g., "synapse") is our unique Identifier.
	if event.Origin == "" {
		log.Printf("[MQTT] Ignored event with empty 'origin' on topic %s", msg.Topic())
		return
	}

	// 3. Pass the event to the State Manager
	// The State Manager will decide if it's a Heartbeat (Info) or an Incident (Critical)
	// We use event.Origin as the Identifier.
	c.stateManager.ProcessEvent(event.Origin, &event)
}

// Disconnect gracefully closes the connection.
func (c *Client) Disconnect() {
	c.internalClient.Disconnect(250)
	log.Println("[MQTT] Disconnected.")
}
