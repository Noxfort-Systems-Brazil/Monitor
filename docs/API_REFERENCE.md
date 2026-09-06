[📚 Documentation Hub](INDEX.md) > **Complete API & Protocol Reference**

---

# 📡 Complete API & Protocol Reference: Noxfort Monitor™

This document formally specifies all communication interfaces of **Noxfort Monitor™ v2.0**, covering the asynchronous **MQTT** ingestion protocol, the **HTTP Telemetry** REST endpoint, the **Authentication & Sessions** subsystem, **Ngrok Tunnel** control APIs, **Database** management, and **Audit Trail** querying.

---

## 1. MQTT Ingestion Protocol

The MQTT broker (Mosquitto) runs by default on `tcp://127.0.0.1:1883` (configurable in `Settings` or `configs/config.yaml`).

### 1.1 Publishing Topics
* **Default Device Topic**: `noxfort/devices/{identifier}/telemetry`
* **Server Subscription**: The Monitor client subscribes using wildcards (`noxfort/devices/+/telemetry` or dedicated topics), processing incoming packets non-blockingly across asynchronous goroutines.

### 1.2 Universal JSON Format (`IncomingEvent`)
Both MQTT messages and HTTP REST requests must transmit the structured JSON body:

```json
{
  "category": "HARDWARE",
  "origin": "sensor-node-tx1",
  "level": "CRITICAL",
  "message": "Overheating detected: Temperature 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Field Specifications:
* **`category`** (*String*, required): `HARDWARE` or `SOFTWARE`. Determines RBAC alert routing rules (Hardware $\rightarrow$ Technicians; Software $\rightarrow$ Programmers).
* **`origin`** (*String*, required): Unique identifier of the edge device (e.g., `carina`, `synapse`, `pump-01`).
* **`level`** (*String*, required): Severity level: `INFO`, `WARNING`, or `CRITICAL`.
  * `INFO` messages containing keep-alive keywords ("*system ok*", "*heartbeat*", "*online*") update the device heartbeat (`last_seen`) but are omitted from permanent persistence to optimize storage.
  * `CRITICAL` messages trigger immediate alert dispatching to all eligible contacts.
* **`message`** (*String*, required): Human-readable operational message.
* **`occurred_at`** (*ISO-8601 Timestamp*, required): Precise timestamp when the event occurred at the origin.

---

## 2. Telemetry Ingestion via HTTP REST

Designed for field nodes (such as **Carina**, **Synapse**, or cURL/Python scripts) operating in external networks where direct MQTT connections to port 1883 are blocked.

### `POST /api/telemetry`
* **Authentication**: Public (exempt from auth middleware to accommodate autonomous field sensors).
* **Content-Type**: `application/json`
* **Body**: Identical to the universal `IncomingEvent` JSON structure.

#### Request Example:
```bash
curl -X POST http://localhost:8080/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Connection pool at 85% capacity.",
    "occurred_at": "2026-09-05T17:00:00Z"
  }'
```

#### Responses:
* **`200 OK`**:
  ```json
  {"status": "received"}
  ```
* **`400 Bad Request`**: Malformed body or missing required fields (`origin`, `level`, or `message`).
* **`405 Method Not Allowed`**: When invoked with HTTP methods other than `POST`.

---

## 3. Authentication & Session Management

Refer to [Security & RBAC](SECURITY.md) for architectural details.

### 3.1 `POST /api/auth/login`
Authenticates the user and issues a session cookie.
* **Format**: Form URL-encoded or JSON (`username`, `password`).
* **Success (`200 OK`)**: Sets the `noxfort_session` cookie with security flags.
  ```json
  {"success": true, "redirect": "/"}
  ```
* **Failure (`401 Unauthorized`)**:
  ```json
  {"success": false, "error": "Invalid credentials"}
  ```

### 3.2 `POST /api/auth/logout`
Invalidates the token in the [`SessionManager`](../internal/security/session.go) and expires the cookie in the client/browser.

### 3.3 `GET /api/auth/status`
Returns the authentication state of the current session.
* **Authenticated Response**:
  ```json
  {
    "authenticated": true,
    "username": "admin",
    "role": "ADMIN"
  }
  ```

---

## 4. Operator Account Management (`/api/users`)

*Requires an authenticated session with `ADMIN` role.*

| Method | Route | Description | Parameters / Body |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/users` | Lists all registered operators. | - |
| `POST` | `/api/users/create` | Creates a new operator or administrator. | `username`, `password`, `role` (`ADMIN` or `OPERATOR`) |
| `POST` | `/api/users/delete` | Deletes the specified user account. | `username` (Query or Form) |

---

## 5. Remote Access & Ngrok Tunnel (`/api/tunnel`)

Refer to [Remote Access via Ngrok](REMOTE_ACCESS.md) for conceptual details.

### 5.1 `GET /api/tunnel/status`
Returns tunnel health and the public telemetry endpoint:
```json
{
  "active": true,
  "public_url": "https://my-monitor.ngrok-free.app",
  "telemetry_url": "https://my-monitor.ngrok-free.app/api/telemetry",
  "domain": "my-monitor.ngrok-free.app",
  "started_at": "2026-09-05T14:00:00Z",
  "binary_found": true,
  "error": ""
}
```

### 5.2 Additional Tunnel Operations:
* `POST /api/tunnel/save`: Saves credentials (`ngrok_auth_token`, `ngrok_domain`, `ngrok_enabled`).
* `POST /api/tunnel/start`: Launches the tunnel process on demand.
* `POST /api/tunnel/stop`: Terminates the external connection.

---

## 6. Database & Server Config (`/api/settings/database`)

Refer to [Database & Dual-Engine Persistence](DATABASE.md).

### 6.1 `GET /api/settings/database/status`
Returns the active database engine and query latency:
```json
{
  "connected": true,
  "type": "postgres",
  "host": "localhost",
  "port": 5432,
  "dbname": "noxfort_database",
  "schema": "schema_monitor",
  "user": "user_monitor",
  "latency_ms": 2,
  "schema_exists": true,
  "version": "PostgreSQL 16.1",
  "server_time": "2026-09-05T17:05:00Z"
}
```

### 6.2 `POST /api/settings/database/test`
Tests connectivity with supplied parameters without altering the active production database connection.

### 6.3 `POST /api/settings/database/save`
Applies new credentials to the [`DBManager`](../internal/storage/db_manager.go). If `migrate=true` is passed, runs data synchronization prior to switching the driver.

### 6.4 `POST /api/settings/database/provision-user`
Provisions a dedicated schema-restricted user using PostgreSQL administrative credentials.

---

## 7. Audit Trail (`/api/audit`)

Refer to [Audit Trail](AUDIT_TRAIL.md).

* `GET /api/audit/security?limit=100`: Security event log history.
* `GET /api/audit/alerts?limit=100`: Alert dispatch history via Email/Telegram with `SENT`/`FAILED` status.
* `GET /api/audit/transitions?limit=100`: Device outage and recovery history recorded by the Watchdog Engine with downtime durations.

---

## 8. Notification Channel Diagnostics

Endpoints used to dispatch verification alerts on demand:
* `POST /settings/test`: Sends a test email using the active SMTP settings.
* `POST /settings/test-telegram`: Sends a MarkdownV2 test message using the configured bot token and chat ID.

---

## 9. Desktop and Browser Controls

* `POST /api/open-external`: Accepts `{"url": "https://..."}` and directs the operating system to open the link in the user's default browser (`xdg-open` on Linux).
* `POST /api/window/toggle-fullscreen`: Toggles the Wails window between fullscreen and normal mode.
* `POST /api/window/exit-fullscreen`: Exits fullscreen mode.

---

### 🔗 Related Documentation
* 🏗️ [System Architecture](../ARCHITECTURE.md) — Data flow overview
* 🗄️ [Database & Persistence](DATABASE.md) — Models and DDL schemas
* 🔐 [Security & RBAC](SECURITY.md) — Authentication mechanisms
* 🌐 [Remote Access](REMOTE_ACCESS.md) — Ngrok tunnel configuration
* 🔍 [Audit Trail](AUDIT_TRAIL.md) — Detailed log schema
* 🧪 [Testing Guide](TESTING.md) — Practical testing examples with cURL and mosquitto
