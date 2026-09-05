[📚 Central de Documentação](INDEX.md) > **Guia de Implantação em Produção**

---

# 🚀 Guia de Implantação em Produção: Noxfort Monitor™

Este documento fornece instruções passo a passo para implantar o **Noxfort Monitor™ v2.0** em ambientes industriais e corporativos de produção, cobrindo o serviço autônomo **Systemd** em modo **Headless**, a instalação simplificada via pacote **Debian (`.deb`)**, a configuração com banco de dados **PostgreSQL** e o proxy reverso **NGINX** com **SSL**.

---

## 1. Opções de Instalação em Produção

Você pode implantar o Noxfort Monitor de duas maneiras principais:
1. **Pacote Nativo `.deb` (Recomendado para estações Ubuntu/Debian)**: Instala binário, dependências, serviço Mosquitto e atalhos de sistema operacional automaticamente.
2. **Serviço Systemd Dedicado em Modo Headless (Recomendado para Servidores/VPS sem interface gráfica)**.

---

## 2. Instalação via Pacote Debian (`.deb`)

Na máquina de desenvolvimento ou esteira de CI/CD:
```bash
# Gera o pacote em build_deb/noxfort-monitor_2.0.1_amd64.deb
make deb
```

No servidor de destino:
```bash
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f # Resolve dependências ausentes, caso necessário
```

Isso instala o software em `/opt/noxfort-monitor/`, cria o link `/usr/local/bin/noxfort-monitor`, instala ícones e inicializa o Mosquitto MQTT via systemd.

---

## 3. Configuração como Serviço Linux (Systemd Headless)

Para servidores dedicados sem interface gráfica (X11 ou Wayland), o binário **deve ser executado com a flag `--headless`**.

### 3.1 Compilar o Binário Otimizado
```bash
make build-linux
# Binário gerado em bin/noxfort-monitor-linux
```

Transfira o binário para o servidor:
```bash
scp bin/noxfort-monitor-linux user@servidor-producao:/usr/local/bin/noxfort-monitor
```

### 3.2 Criar Usuário do Sistema e Permissões
```bash
sudo useradd -m -s /bin/bash noxfort
sudo chown noxfort:noxfort /usr/local/bin/noxfort-monitor
sudo chmod +x /usr/local/bin/noxfort-monitor
```

### 3.3 Criar o Arquivo de Serviço Systemd
Crie `/etc/systemd/system/noxfort-monitor.service`:

```ini
[Unit]
Description=Noxfort Monitor Server (Headless Production Service)
After=network.target mosquitto.service postgresql.service
Wants=mosquitto.service

[Service]
Type=simple
User=noxfort
Group=noxfort

# Execução obrigatória com --headless em servidores sem display gráfico
ExecStart=/usr/local/bin/noxfort-monitor --headless

Restart=on-failure
RestartSec=5

# Diretório de trabalho onde se localiza o .env de produção
WorkingDirectory=/home/noxfort

# Variáveis de ambiente fundamentais
Environment="PORT=8080"
EnvironmentFile=-/home/noxfort/.env

# Limites de recursos (opcional)
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

### 3.4 Configurar o Arquivo `.env` do Usuário de Produção
Crie `/home/noxfort/.env`:
```ini
MONITOR_ADMIN_USER=admin_corporativo
MONITOR_ADMIN_PASSWORD=SenhaForteCriptografada123!
PORT=8080
```
Proteja o arquivo contra leitura não autorizada:
```bash
sudo chmod 600 /home/noxfort/.env
sudo chown noxfort:noxfort /home/noxfort/.env
```

### 3.5 Habilitar e Iniciar o Serviço
```bash
sudo systemctl daemon-reload
sudo systemctl enable noxfort-monitor
sudo systemctl start noxfort-monitor
```

Verifique o status e logs em tempo real:
```bash
sudo systemctl status noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 4. Configuração do Banco de Dados PostgreSQL em Produção

Para infraestruturas de alta concorrência e conformidade:

1. **Instalar PostgreSQL 14+**:
   ```bash
   sudo apt-get install -y postgresql postgresql-contrib
   ```
2. **Criar Banco e Usuário com Acesso ao Schema**:
   O Monitor pode provisionar as tabelas automaticamente. Basta garantir que o banco exista:
   ```sql
   CREATE DATABASE banco_de_dados_noxfort;
   CREATE USER user_monitor WITH PASSWORD 'sua_senha_segura';
   GRANT ALL PRIVILEGES ON DATABASE banco_de_dados_noxfort TO user_monitor;
   ```
3. **Conectar pelo Painel**:
   Acesse a tela `/server` ou envie uma requisição a `/api/settings/database/save` apontando para o PostgreSQL. O sistema criará o schema `schema_monitor`, todas as tabelas e índices automaticamente. Consulte [Banco de Dados & Dual-Engine](DATABASE.md).

---

## 5. Proxy Reverso com NGINX e SSL

Embora o Noxfort Monitor possua autenticação e proteção RBAC embutida ([`AuthMiddleware`](SECURITY.md)), colocar um proxy reverso NGINX na frente do serviço garante terminação SSL e proteção contra ataques de negação de serviço.

### Exemplo de Configuração NGINX:
Crie `/etc/nginx/sites-available/noxfort-monitor`:

```nginx
server {
    listen 80;
    server_name monitor.suaempresa.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitor.suaempresa.com;

    ssl_certificate /etc/letsencrypt/live/monitor.suaempresa.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.suaempresa.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Ingestão de telemetria e console web
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

Ative a configuração:
```bash
sudo ln -s /etc/nginx/sites-available/noxfort-monitor /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 6. Acesso Externo via Túnel Ngrok (Sem Abrir Portas no Roteador)

Caso o servidor esteja atrás de firewalls industriais sem IP público ou domínio DNS, ative o túnel seguro Ngrok:
* Siga as instruções do guia [Acesso Remoto & Túnel Ngrok](REMOTE_ACCESS.md).
* O túnel abrirá uma conexão de saída com domínio estático, permitindo que agentes remotos (Synapse e Carina) entreguem telemetria em `https://seu-dominio.ngrok-free.app/api/telemetry`.

---

### 🔗 Documentos Relacionados
* 🖥️ [Aplicação Desktop](DESKTOP_APP.md) — Operação local com interface gráfica Wails
* 🗄️ [Banco de Dados & Persistência](DATABASE.md) — Configuração e migração para PostgreSQL
* 🔐 [Segurança e RBAC](SECURITY.md) — Configuração de contas e superusuário
* 🌐 [Acesso Remoto](REMOTE_ACCESS.md) — Túnel seguro para conexão de nós externos
* 📡 [Referência de APIs](API_REFERENCE.md) — Especificação de endpoints de monitoramento
* 🧪 [Guia de Testes](TESTING.md) — Validação de funcionamento após o deploy
