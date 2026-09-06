[📚 Documentation Hub](INDEX.md) > **Production Deployment Guide**

---

# 🚀 Production Deployment Guide: Noxfort Monitor™

This document provides step-by-step instructions for deploying **Noxfort Monitor™ v2.0** in production industrial and enterprise environments, covering standalone **Systemd** service setup in **Headless** mode, streamlined **Debian package (`.deb`)** installation, **PostgreSQL** database configuration, and **NGINX** reverse proxy deployment with **SSL**.

---

## 1. Production Deployment Options

You can deploy Noxfort Monitor in two primary ways:
1. **Native `.deb` Package (Recommended for Ubuntu/Debian workstations)**: Automatically installs the binary, dependencies, Mosquitto service, desktop integration, and system shortcuts.
2. **Dedicated Systemd Service in Headless Mode (Recommended for headless servers and cloud VPS instances)**.

---

## 2. Installation via Debian Package (`.deb`)

On your build machine or CI/CD runner:
```bash
# Generates the package in build_deb/noxfort-monitor_2.0.1_amd64.deb
make deb
```

On the target host:
```bash
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f # Resolves missing dependencies if needed
```

This installs the software to `/opt/noxfort-monitor/`, symlinks `/usr/local/bin/noxfort-monitor`, installs desktop application icons, and enables Mosquitto MQTT via systemd.

---

## 3. Linux Service Configuration (Headless Systemd)

For dedicated servers without a display server (X11 or Wayland), the binary **must be launched with the `--headless` flag**.

### 3.1 Build the Production Binary
```bash
make build-linux
# Binary compiled to bin/noxfort-monitor-linux
```

Copy the binary to the server:
```bash
scp bin/noxfort-monitor-linux user@production-server:/usr/local/bin/noxfort-monitor
```

### 3.2 Create Dedicated System User and Permissions
```bash
sudo useradd -m -s /bin/bash noxfort
sudo chown noxfort:noxfort /usr/local/bin/noxfort-monitor
sudo chmod +x /usr/local/bin/noxfort-monitor
```

### 3.3 Create the Systemd Unit File
Create `/etc/systemd/system/noxfort-monitor.service`:

```ini
[Unit]
Description=Noxfort Monitor Server (Headless Production Service)
After=network.target mosquitto.service postgresql.service
Wants=mosquitto.service

[Service]
Type=simple
User=noxfort
Group=noxfort

# Headless flag is required on servers without a graphical display
ExecStart=/usr/local/bin/noxfort-monitor --headless

Restart=on-failure
RestartSec=5

# Working directory containing production .env
WorkingDirectory=/home/noxfort

# Environment variables
Environment="PORT=8080"
EnvironmentFile=-/home/noxfort/.env

# Resource limits (optional)
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

### 3.4 Configure the Production `.env` File
Create `/home/noxfort/.env`:
```ini
MONITOR_ADMIN_USER=corporate_admin
MONITOR_ADMIN_PASSWORD=StrongEncryptedPassword123!
PORT=8080
```
Restrict file permissions:
```bash
sudo chmod 600 /home/noxfort/.env
sudo chown noxfort:noxfort /home/noxfort/.env
```

### 3.5 Enable and Start the Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable noxfort-monitor
sudo systemctl start noxfort-monitor
```

Inspect service health and real-time logs:
```bash
sudo systemctl status noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 4. PostgreSQL Configuration in Production

For high-concurrency environments and regulatory compliance:

1. **Install PostgreSQL 14+**:
   ```bash
   sudo apt-get install -y postgresql postgresql-contrib
   ```
2. **Create Database and User with Schema Access**:
   The Monitor provisions its own tables automatically. Ensure the target database exists:
   ```sql
   CREATE DATABASE noxfort_database;
   CREATE USER user_monitor WITH PASSWORD 'your_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE noxfort_database TO user_monitor;
   ```
3. **Connect via Dashboard**:
   Access `/server` or submit a `POST` request to `/api/settings/database/save` targeting PostgreSQL. The system automatically creates `schema_monitor`, tables, and indices. See [Database & Dual-Engine Persistence](DATABASE.md).

---

## 5. NGINX Reverse Proxy with SSL Termination

While Noxfort Monitor includes built-in authentication and RBAC via [`AuthMiddleware`](SECURITY.md), placing an NGINX reverse proxy in front provides SSL termination and DDoS mitigation.

### Example NGINX Configuration:
Create `/etc/nginx/sites-available/noxfort-monitor`:

```nginx
server {
    listen 80;
    server_name monitor.yourcompany.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitor.yourcompany.com;

    ssl_certificate /etc/letsencrypt/live/monitor.yourcompany.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.yourcompany.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Telemetry ingestion and web console
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

Enable the configuration:
```bash
sudo ln -s /etc/nginx/sites-available/noxfort-monitor /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 6. Remote Access via Ngrok Tunnel (No Inbound Port Forwarding)

When the server operates behind industrial firewalls or carrier-grade NAT (CGNAT) without public static IPs or DNS:
* Follow the instructions in [Remote Access & Ngrok Tunnel](REMOTE_ACCESS.md).
* The tunnel establishes an outbound connection with a static domain, allowing remote agents (Synapse and Carina) to post telemetry to `https://your-domain.ngrok-free.app/api/telemetry`.

---

### 🔗 Related Documentation
* 🖥️ [Desktop Application](DESKTOP_APP.md) — Local operations with Wails GUI
* 🗄️ [Database & Persistence](DATABASE.md) — Configuration and migration to PostgreSQL
* 🔐 [Security & RBAC](SECURITY.md) — Account configuration and superuser bootstrapping
* 🌐 [Remote Access](REMOTE_ACCESS.md) — Secure tunneling for external edge nodes
* 📡 [API Reference](API_REFERENCE.md) — Monitoring endpoints specification
* 🧪 [Testing Guide](TESTING.md) — Post-deployment verification
