[📚 Central de Documentação](INDEX.md) > **Notas de Pesquisa & Decisões Técnicas**

---

# 🔬 Notas de Pesquisa & Decisões Técnicas: Noxfort Monitor™

Este documento serve como repositório de decisões de arquitetura de software, benchmarks e direções de pesquisa para o futuro do **Noxfort Monitor™**.

---

## 1. Persistência: Da Dependência CGO à Arquitetura Dual-Engine

### Contexto
Versões anteriores do Monitor utilizavam exclusivamente `mattn/go-sqlite3`, exigindo compilador C (CGO) para construir o binário. Isso gerava grande atrito na compilação cruzada e na distribuição de pacotes.

### Decisão
1. **Migração para Pure-Go SQLite**: Adotamos `modernc.org/sqlite` para permitir compilação Go 100% pura no modo embarcado.
2. **Evolução para Dual-Engine (SQLite + PostgreSQL)**: Em ambientes industriais de grande porte, múltiplos operadores acessam o dashboard simultaneamente, e concorrência massiva de escrita em SQLite gerava contenção de locks (`database is locked`).
3. **Hot-Reload Sem Queda**: Implementamos o [`DBManager`](DATABASE.md) para permitir alternar entre SQLite e PostgreSQL a qualquer momento via interface gráfica, com migração automática dos registros, sem derrubar a ingestão de telemetria.

---

## 2. Interface Desktop: De Systray Simples para Wails v2

### Contexto
A primeira versão utilizava apenas um ícone na barra de tarefas (`getlantern/systray`) que abria o navegador padrão do sistema. Usuários reportaram problemas de concorrência com sessões de outros sistemas abertas no mesmo navegador, falta de atalhos dedicados e ausência de uma experiência desktop unificada.

### Decisão
Migramos para **Wails v2** ([`docs/DESKTOP_APP.md`](DESKTOP_APP.md)):
* **WebKitGTK Nativo**: Rendição rápida e consistente de HTML5/Bootstrap sem o consumo de memória de navegadores pesados baseados em Chromium.
* **Ciclo de Vida Integrado**: O systray foi reescrito para viver dentro do loop de eventos do Wails, permitindo minimizar a janela ao fechar (`HideWindowOnClose`) e restaurar com um clique.
* **Isolamento de Sessões**: O interceptador de cabeçalhos sincroniza o cookie `noxfort_session` na memória do desktop.
* **Compatibilidade Headless**: Adição do modo `--headless` para permitir que o exato mesmo binário rode como serviço de servidor Linux sem falhas de display gráfico.

---

## 3. Conexão WAN: Integração de Túnel Reverso (Ngrok)

### Contexto
Sensores em filiais, nós em veículos e agentes de borda (como **Synapse** e **Carina**) operam fora da rede interna do servidor do Monitor. Abrir portas em roteadores industriais é proibido pela maioria das políticas de segurança cibernética (Norma ISA/IEC 62443).

### Decisão
Incorporamos o subsistema [`internal/tunnel`](REMOTE_ACCESS.md):
* O Monitor estabelece um túnel outbound seguro via TLS com a nuvem do Ngrok.
* Clientes externos enviam dados para um domínio estável (ex: `https://seu-monitor.ngrok-free.app/api/telemetry`).
* A interface do usuário adapta automaticamente os links sugeridos de telemetria.

---

## 4. Próximos Passos no Roadmap

### 4.1 Expansão com gRPC para Federação de Monitores
Embora o MQTT e HTTP REST atendam com folga à telemetria de sensores, a federação entre múltiplos servidores Noxfort Monitor (clustering ativo-passivo ou hierarquia matriz-filiais) se beneficiará de buffers binários **gRPC / Protocol Buffers**:
* Redução drástica de overhead de rede em links de satélite.
* Schemas tipados imutáveis para sincronização de audit logs.

### 4.2 Detecção de Anomalias Preditivas
Atualmente os alertas disparam por regras e limites estáticos (ex: `temperatura > 80°C`). Estamos investigando modelos leves de previsão de séries temporais em Go (ex: médias móveis exponenciais e ONNX runtime embarcado) para detectar tendências de falha antes que os limites operacionais sejam rompidos.

---

### 🔗 Documentos Relacionados
* 🏗️ [Arquitetura Geral](../ARCHITECTURE.md) — Visão macro do sistema
* 🗄️ [Banco de Dados & Persistência](DATABASE.md) — Implementação do Dual-Engine
* 🖥️ [Aplicação Desktop](DESKTOP_APP.md) — Detalhes da arquitetura Wails
* 🌐 [Acesso Remoto](REMOTE_ACCESS.md) — Racional do túnel reverso
