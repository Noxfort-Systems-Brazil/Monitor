[📚 Central de Documentação](INDEX.md) > **Referência Completa de APIs & Protocolos**

---

# 📡 Referência Completa de APIs & Protocolos: Noxfort Monitor™

Este documento especifica formalmente todas as interfaces de comunicação do **Noxfort Monitor™ v2.0**, cobrindo o protocolo de ingestão assíncrona **MQTT**, o endpoint REST de **Telemetria HTTP**, o subsistema de **Autenticação & Sessões**, as APIs de controle do **Túnel Ngrok**, gerenciamento de **Banco de Dados** e extração da **Trilha de Auditoria**.

---

## 1. Protocolo de Ingestão MQTT

O broker MQTT (Mosquitto) opera por padrão em `tcp://127.0.0.1:1883` (configurável em `Settings` ou `configs/config.yaml`).

### 1.1 Tópicos de Publicação
* **Tópico Padrão de Dispositivo**: `noxfort/devices/{identifier}/telemetry`
* **Subscrição do Servidor**: O cliente do Monitor subscreve via wildcards (`noxfort/devices/+/telemetry` ou tópicos dedicados), processando pacotes de forma não-bloqueante em goroutines assíncronas.

### 1.2 O Formato Universal JSON (`IncomingEvent`)
Tanto mensagens MQTT quanto requisições HTTP REST devem enviar o corpo JSON estruturado:

```json
{
  "category": "HARDWARE",
  "origin": "sensor-node-tx1",
  "level": "CRITICAL",
  "message": "Superaquecimento detectado: Temperatura 95°C",
  "occurred_at": "2026-09-05T14:30:00Z"
}
```

#### Especificação dos Campos:
* **`category`** (*String*, obrigatório): `HARDWARE` ou `SOFTWARE`. Determina a regra de roteamento RBAC dos alertas (Hardware $\rightarrow$ Técnicos; Software $\rightarrow$ Programadores).
* **`origin`** (*String*, obrigatório): Identificador único do dispositivo de borda (ex: `carina`, `synapse`, `pump-01`).
* **`level`** (*String*, obrigatório): Severidade: `INFO`, `WARNING` ou `CRITICAL`.
  * Mensagens `INFO` com palavras-chave de keep-alive ("*system ok*", "*heartbeat*", "*online*") atualizam o batimento cardíaco (`last_seen`) mas são descartadas da persistência para economizar armazenamento.
  * Mensagens `CRITICAL` disparam despacho imediato a todos os contatos elegíveis.
* **`message`** (*String*, obrigatório): Texto legível para operadores humanos.
* **`occurred_at`** (*Timestamp ISO-8601*, obrigatório): Carimbo de tempo do momento exato do evento na origem.

---

## 2. Ingestão de Telemetria via HTTP REST

Projetada para nós de campo (como **Carina**, **Synapse** ou scripts cURL/Python) que operam em redes externas onde conexões diretas MQTT com a porta 1883 estão bloqueadas.

### `POST /api/telemetry`
* **Autenticação**: Pública (isenta de middleware para permitir sensores autônomos de campo).
* **Content-Type**: `application/json`
* **Corpo**: Estrutura idêntica ao JSON universal `IncomingEvent`.

#### Exemplo de Requisição:
```bash
curl -X POST http://localhost:8080/api/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "category": "SOFTWARE",
    "origin": "synapse-core",
    "level": "WARNING",
    "message": "Pool de conexões em 85% de uso.",
    "occurred_at": "2026-09-05T17:00:00Z"
  }'
```

#### Respostas:
* **`200 OK`**:
  ```json
  {"status": "received"}
  ```
* **`400 Bad Request`**: Corpo inválido ou ausência de campos obrigatórios (`origin`, `level` ou `message`).
* **`405 Method Not Allowed`**: Quando chamado com métodos diferentes de `POST`.

---

## 3. Autenticação & Gestão de Sessões

Consulte o documento [Segurança e RBAC](SECURITY.md) para detalhes da arquitetura.

### 3.1 `POST /api/auth/login`
Autentica o usuário e emite o cookie de sessão.
* **Formato**: Form URL-encoded ou JSON (`username`, `password`).
* **Sucesso (`200 OK`)**: Define o cookie `noxfort_session` com flags de segurança.
  ```json
  {"success": true, "redirect": "/"}
  ```
* **Falha (`401 Unauthorized`)**:
  ```json
  {"success": false, "error": "Credenciais inválidas"}
  ```

### 3.2 `POST /api/auth/logout`
Invalida o token no [`SessionManager`](../internal/security/session.go) e expira o cookie no navegador/desktop.

### 3.3 `GET /api/auth/status`
Retorna o estado de autenticação da sessão atual.
* **Resposta Autenticada**:
  ```json
  {
    "authenticated": true,
    "username": "admin",
    "role": "ADMIN"
  }
  ```

---

## 4. Gestão de Contas de Operadores (`/api/users`)

*Requer sessão autenticada com papel `ADMIN`.*

| Método | Rota | Descrição | Parâmetros / Corpo |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/users` | Lista todos os operadores cadastrados. | - |
| `POST` | `/api/users/create` | Cria novo operador ou administrador. | `username`, `password`, `role` (`ADMIN` ou `OPERATOR`) |
| `POST` | `/api/users/delete` | Exclui a conta informada. | `username` (Query ou Form) |

---

## 5. Acesso Remoto & Túnel Ngrok (`/api/tunnel`)

Consulte o documento [Acesso Remoto via Ngrok](REMOTE_ACCESS.md) para detalhes conceituais.

### 5.1 `GET /api/tunnel/status`
Retorna a integridade do túnel e o endereço público de telemetria:
```json
{
  "active": true,
  "public_url": "https://meu-monitor.ngrok-free.app",
  "telemetry_url": "https://meu-monitor.ngrok-free.app/api/telemetry",
  "domain": "meu-monitor.ngrok-free.app",
  "started_at": "2026-09-05T14:00:00Z",
  "binary_found": true,
  "error": ""
}
```

### 5.2 Outras Operações de Túnel:
* `POST /api/tunnel/save`: Salva credenciais (`ngrok_auth_token`, `ngrok_domain`, `ngrok_enabled`).
* `POST /api/tunnel/start`: Dispara o processo do túnel sob demanda.
* `POST /api/tunnel/stop`: Interrompe a conexão externa.

---

## 6. Banco de Dados & Server Config (`/api/settings/database`)

Consulte o documento [Banco de Dados & Dual-Engine](DATABASE.md).

### 6.1 `GET /api/settings/database/status`
Retorna o motor em execução e latência de consulta:
```json
{
  "connected": true,
  "type": "postgres",
  "host": "localhost",
  "port": 5432,
  "dbname": "banco_de_dados_noxfort",
  "schema": "schema_monitor",
  "user": "user_monitor",
  "latency_ms": 2,
  "schema_exists": true,
  "version": "PostgreSQL 16.1",
  "server_time": "2026-09-05T17:05:00Z"
}
```

### 6.2 `POST /api/settings/database/test`
Testa conectividade com parâmetros fornecidos sem alterar a conexão de produção ativa.

### 6.3 `POST /api/settings/database/save`
Aplica novas credenciais no [`DBManager`](../internal/storage/db_manager.go). Se o parâmetro `migrate=true` for enviado, executa a sincronização de dados antes da troca de driver.

### 6.4 `POST /api/settings/database/provision-user`
Cria usuário restrito com permissões restritas ao schema usando credenciais administrativas do PostgreSQL.

---

## 7. Trilha de Auditoria (`/api/audit`)

Consulte o documento [Trilha de Auditoria](AUDIT_TRAIL.md).

* `GET /api/audit/security?limit=100`: Histórico de eventos de segurança.
* `GET /api/audit/alerts?limit=100`: Histórico de despacho de alertas via Email/Telegram com status `SENT`/`FAILED`.
* `GET /api/audit/transitions?limit=100`: Histórico de detecção de quedas e recuperações do Watchdog Engine com duração do downtime.

---

## 8. Diagnóstico de Canais de Alerta

Endpoints utilizados para envio de alertas de verificação sob demanda:
* `POST /settings/test`: Envia email de teste utilizando as configurações SMTP ativas.
* `POST /settings/test-telegram`: Envia mensagem MarkdownV2 de teste utilizando o token do bot e chat ID configurados.

---

## 9. Controles do Desktop e Navegador

* `POST /api/open-external`: Recebe `{"url": "https://..."}` e instrui o sistema operacional a abrir o link no navegador padrão do usuário (`xdg-open` no Linux).
* `POST /api/window/toggle-fullscreen`: Alterna a janela Wails entre tela cheia e tamanho normal.
* `POST /api/window/exit-fullscreen`: Sai do modo de tela cheia.

---

### 🔗 Documentos Relacionados
* 🏗️ [Arquitetura Geral](../ARCHITECTURE.md) — Visão dos fluxos de dados
* 🗄️ [Banco de Dados & Persistência](DATABASE.md) — Modelos e DDL
* 🔐 [Segurança e RBAC](SECURITY.md) — Mecanismos de autenticação
* 🌐 [Acesso Remoto](REMOTE_ACCESS.md) — Configuração do túnel Ngrok
* 🔍 [Trilha de Auditoria](AUDIT_TRAIL.md) — Estrutura detalhada dos logs
* 🧪 [Guia de Testes](TESTING.md) — Exemplos práticos de teste com cURL e mosquitto
