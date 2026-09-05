[📚 Central de Documentação](INDEX.md) > **Trilha de Auditoria & Observabilidade**

---

# 🔍 Trilha de Auditoria & Observabilidade: Noxfort Monitor™

Este documento descreve o subsistema de conformidade e trilha de auditoria (*Audit Trail*) do **Noxfort Monitor™ v2.0**, projetado para atender a normas regulatórias industriais, auditoria de segurança de acessos, verificação de entrega de alertas (SLA) e rastreabilidade de disponibilidade de hardware.

---

## 1. As Três Frentes de Auditoria

O Noxfort Monitor segrega os registros de auditoria em três fluxos independentes e imutáveis ([`internal/domain/audit.go`](../internal/domain/audit.go)):

```mermaid
graph TD
    subgraph "Fontes de Eventos"
        Auth[SecurityManager / AuthMiddleware]
        Alerts[AlertService / Dispatcher]
        Watchdog[Engine / Watchdog]
    end

    subgraph "Camada de Auditoria (AuditRepository)"
        SecLog[1. SecurityAuditLog]
        AlertLog[2. AlertDispatchLog]
        TransLog[3. DeviceStateTransition]
    end

    subgraph "Persistência (Postgres / SQLite)"
        DB[(security_audit_logs / alert_dispatch_logs / device_state_transitions)]
    end

    subgraph "Visualização & Análise"
        UIWeb[Tela de Auditoria /audit]
        APIEndpoints[APIs /api/audit/*]
    end

    Auth -->|Login / Falha / Configs| SecLog
    Alerts -->|Email / Telegram / Status| AlertLog
    Watchdog -->|Queda / Recuperação / Downtime| TransLog

    SecLog --> DB
    AlertLog --> DB
    TransLog --> DB

    DB --> UIWeb
    DB --> APIEndpoints
```

---

## 2. Detalhamento dos Modelos de Log

### 2.1 Auditoria de Segurança (`SecurityAuditLog`)
Rastreia todas as ações administrativas e autenticações com o IP do cliente e data/hora UTC:

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `int64` | Identificador sequencial único. |
| `created_at` | `time.Time` | Timestamp de registro da ação. |
| `username` | `string` | Nome do usuário autor da ação (ou informado na tentativa de login). |
| `action` | `string` | Identificador padronizado da ação (veja tabela abaixo). |
| `details` | `string` | Descrição contextual (ex: configurações alteradas, ID do dispositivo excluído). |
| `ip_address` | `string` | Endereço IP do requisitante. |

#### Ações Comuns de Segurança:
* `AUTH_LOGIN_SUCCESS`: Login com credenciais válidas.
* `AUTH_LOGIN_FAILED`: Tentativa de login com senha ou usuário incorreto.
* `AUTH_LOGOUT`: Encerramento voluntário de sessão.
* `USER_CREATED` / `USER_DELETED`: Gestão de contas de operadores.
* `SETTINGS_UPDATED`: Alteração de canais de notificação (SMTP, Telegram, Ngrok).
* `DATABASE_SWITCHED`: Chaveamento dinâmico entre SQLite e PostgreSQL.
* `DEVICE_DELETED`: Remoção de um sistema da lista de monitoramento.

### 2.2 Auditoria de Despacho de Alertas (`AlertDispatchLog`)
Garante rastreabilidade de entrega para conformidade com **SLAs de resposta a incidentes**:

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `int64` | Identificador sequencial. |
| `telemetry_id` | `*int64` | ID do evento de telemetria que originou o alerta (opcional). |
| `channel` | `string` | Canal de notificação: `EMAIL` ou `TELEGRAM`. |
| `recipient` | `string` | Endereço de destino (email do operador ou `chat_id` do Telegram). |
| `role` | `string` | Papel do destinatário (`TECHNICIAN`, `PROGRAMMER`, `ADMIN`). |
| `status` | `string` | `SENT` (entregue com sucesso), `FAILED` (falha na API/SMTP) ou `SKIPPED`. |
| `error_reason` | `string` | Mensagem de erro retornada pelo servidor SMTP ou API do Telegram caso `FAILED`. |
| `dispatched_at`| `time.Time` | Momento exato em que a notificação foi enviada. |

### 2.3 Transições de Estado de Hardware (`DeviceStateTransition`)
Calcula e registra formalmente o tempo de inatividade (*downtime*) de cada equipamento:

| Campo | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `int64` | Identificador sequencial. |
| `device_identifier` | `string` | Identificador do sistema de origem (ex: `synapse`, `pump-01`). |
| `previous_state` | `string` | Estado anterior: `ONLINE` ou `OFFLINE`. |
| `new_state` | `string` | Novo estado detectado: `OFFLINE` ou `ONLINE`. |
| `duration_offline_sec` | `int64` | Tempo total (em segundos) em que o sistema permaneceu mudo até se recuperar. |
| `transition_at` | `time.Time` | Momento exato da transição de estado. |

---

## 3. Visualização na Interface Web (`/audit`)

A interface acessível em `/audit` organiza os registros em abas visuais de fácil consulta para operadores e auditores externos:
* **Filtros Visuais**: Cores indicativas para eventos de risco (`FAILED` em vermelho, `SENT` em verde, `OFFLINE` em laranja).
* **Cálculo de Downtime**: Formatação legível da duração de paradas (ex: `12m 45s`).
* **Proteção de Acesso**: Apenas usuários autenticados com papéis autorizados têm acesso à tela.

---

## 4. Endpoints da API REST de Auditoria

Todas as rotas de auditoria retornam dados estruturados em JSON para integração com SIEMs ou ferramentas externas de Business Intelligence:

| Método | Endpoint | Parâmetros | Descrição |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/audit/security` | `limit` (padrão: 100) | Retorna os últimos logs de segurança e autenticação. |
| `GET` | `/api/audit/alerts` | `limit` (padrão: 100) | Retorna o histórico de notificações e status de entrega. |
| `GET` | `/api/audit/transitions` | `limit` (padrão: 100) | Retorna o histórico de disponibilidade e downtime dos dispositivos. |

---

### 🔗 Documentos Relacionados
* 🏗️ [Arquitetura Geral](../ARCHITECTURE.md) — O motor Watchdog e State Manager
* 🔐 [Segurança e RBAC](SECURITY.md) — Eventos de autenticação e proteção de rotas
* 🗄️ [Banco de Dados & Persistência](DATABASE.md) — Tabelas e índices de auditoria no Postgres e SQLite
* 📡 [Referência de APIs](API_REFERENCE.md) — Especificação técnica dos endpoints JSON
* 🖥️ [Aplicação Desktop](DESKTOP_APP.md) — Visualização integrada no desktop
