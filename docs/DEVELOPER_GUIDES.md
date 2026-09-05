[📚 Central de Documentação](INDEX.md) > **Guia Avançado do Desenvolvedor**

---

# 👨‍💻 Guia Avançado do Desenvolvedor: Noxfort Monitor™

Este documento fornece diretrizes de engenharia de software para compilar, manter e estender o **Noxfort Monitor™ v2.0**, cobrindo o setup local, convenções arquiteturais **SOLID**, padrões de repositório com **hot-reload**, injeção de dependência e modelos de concorrência.

---

## 1. Ambiente Local & Compilação

### 1.1 Pré-Requisitos do Sistema (Ubuntu / Debian)
* **Go Toolkit**: Versão 1.22 ou superior.
* **Bibliotecas C/GTK & WebKit**: Necessárias para o runtime gráfico do Wails v2 e systray:
  ```bash
  sudo apt-get update
  sudo apt-get install -y \
      libgtk-3-dev \
      libwebkit2gtk-4.1-dev \
      libappindicator3-dev || sudo apt-get install -y libayatana-appindicator3-dev
  ```
* **Broker Mosquitto Local**:
  ```bash
  make broker-install
  make broker-start
  ```

### 1.2 Comandos Fundamentais do Makefile

| Comando | Descrição |
| :--- | :--- |
| `make build` | Compila o binário desktop Wails com tags de produção (`bin/noxfort-monitor`). |
| `make run` | Executa a aplicação completa em modo de desenvolvimento com interface gráfica. |
| `make run-headless` | Executa o servidor em modo Headless (sem abrir janela desktop). |
| `make test` | Executa toda a suíte de testes automatizados com relatório detalhado. |
| `make deb` | Gera o pacote de instalação Debian (`.deb`) para distribuição. |
| `make broker-start` | Inicia o broker MQTT nativo Mosquitto. |
| `make broker-stop` | Para o broker MQTT. |
| `make broker-status`| Inspeciona o estado do serviço Mosquitto. |

---

## 2. Padrões de Projeto & Arquitetura

O Noxfort Monitor segue rigorosamente os princípios **SOLID** e a arquitetura em camadas orientada a eventos.

### 2.1 Raiz de Composição (Composition Root)
Toda a instanciação de objetos e amarração de dependências ocorre explicitamente em [`cmd/server/main.go`](../cmd/server/main.go).
* **Zero Variáveis Globais**: Nenhum pacote ou módulo utiliza variáveis de estado globais ou pacotes com `init()` mágicos.
* **Injeção de Dependências (DI)**: Construtores recebem interfaces, garantindo desacoplamento e testabilidade com mocks em memória.

### 2.2 O Padrão de Repositório com Hot-Reload
A camada de persistência em [`internal/storage`](../internal/storage) isola o mecanismo de armazenamento através de interfaces do [`internal/domain`](../internal/domain):
* Para suportar a troca de banco de dados em tempo de execução sem reiniciar o servidor, os repositórios implementam a interface `ReloadableRepository`:
  ```go
  type ReloadableRepository interface {
      SetDB(db *sql.DB, driver string)
  }
  ```
* Quando o usuário alterna entre SQLite e PostgreSQL, o [`DBManager`](../internal/storage/db_manager.go) notifica todos os repositórios cadastrados através de `SetDB`.

### 2.3 Boas Práticas de Concorrência
* **Rastreamento de Estados do Watchdog**: O estado de presença de cada equipamento é controlado em memória pela estrutura [`SystemStatusTracker`](../internal/monitor/tracker.go), protegida por `sync.RWMutex` para evitar condições de corrida (*race conditions*).
* **Despacho Assíncrono de Alertas**: A emissão de notificações via Email (SMTP) e Telegram é executada em goroutines independentes dentro de [`AlertService.TriggerAlert()`](../internal/monitor/alerts.go). Uma lentidão eventual de um servidor de email externo nunca bloqueia a fila de eventos do broker MQTT.

---

## 3. Como Estender o Sistema

### 3.1 Adicionar um Novo Canal de Notificação (ex: Discord, Slack ou WhatsApp)
1. Crie o novo canal implementando a interface `NotificationChannel` definida em [`internal/monitor/channel.go`](../internal/monitor/channel.go):
   ```go
   type NotificationChannel interface {
       Name() string
       Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error
       Recipient(contact *domain.Contact) string
   }
   ```
2. Adicione os campos necessários na entidade [`Settings`](../internal/domain/settings.go) (ex: `DiscordWebhookURL`).
3. Adicione o novo canal ao construtor do `AlertService` em `cmd/server/main.go`:
   ```go
   alertService := monitor.NewAlertService(contactRepo, settingsRepo, emailChan, telegramChan, discordChan)
   ```

### 3.2 Adicionar uma Nova Tabela / Entidade no Banco
1. Declare o struct e a interface do repositório em `internal/domain`.
2. Implemente o repositório em `internal/storage` implementando `ReloadableRepository` e utilizando [`storage.AdaptQuery`](../internal/storage/query_adapter.go) para suportar SQLite e PostgreSQL simultaneamente.
3. Adicione a criação da tabela no método `NewDatabase` em `database.go` (para SQLite).
4. Adicione o DDL correspondente no método `InitPostgresSchema` em `postgres_schema.go` (para PostgreSQL).
5. Se a tabela possuir dados que devam ser preservados ao trocar de banco, adicione a rotina de cópia em `migrator.go`.

---

### 🔗 Documentos Relacionados
* 🏗️ [Arquitetura do Sistema](../ARCHITECTURE.md) — Filosofia e fluxo de eventos
* 🗄️ [Banco de Dados & Persistência](DATABASE.md) — Detalhes do DBManager e QueryAdapter
* 🖥️ [Aplicação Desktop](DESKTOP_APP.md) — Ciclo de vida da janela e SingleInstance
* 🧪 [Guia de Testes](TESTING.md) — Padrões de testes unitários com mocks
* 🤝 [Guia de Contribuição](../CONTRIBUTING.md) — Processo de Pull Request e estilo de código
