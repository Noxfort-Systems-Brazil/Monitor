[📚 Central de Documentação](docs/INDEX.md) > **Guia de Contribuição**

---

# 🤝 Contribuindo para o Noxfort Monitor™

Agradecemos o seu interesse em contribuir para o **Noxfort Monitor™**. Este projeto é mantido com altos padrões de integridade arquitetural, princípios **SOLID** e cobertura de testes automatizados.

---

## 1. Código de Conduta

Ao participar deste projeto, espera-se que todos os contribuidores tratem a comunidade com respeito, mantenham uma comunicação profissional e construtiva, e zelem pela qualidade e segurança do código.

---

## 2. Como Posso Contribuir?

### 2.1 Reportando Bugs
* **Pesquise nas issues**: Verifique se o problema já foi reportado anteriormente.
* **Informações do Ambiente**: Sempre forneça a versão do Go (`go version`), distribuição Linux (`lsb_release -a`), versão do Mosquitto e se o bug ocorre em modo Desktop (Wails) ou modo Headless (`--headless`).
* **Logs Relevantes**: Anexe os logs do console ou execute com `journalctl -u noxfort-monitor -f`.

### 2.2 Sugerindo Melhorias & Novas Funcionalidades
* Abra uma issue descrevendo o caso de uso industrial.
* Se a melhoria envolver novas tabelas ou bancos, consulte [Banco de Dados & Persistência](docs/DATABASE.md).
* Se a melhoria envolver novas rotas de comunicação, consulte [Referência de APIs](docs/API_REFERENCE.md).

### 2.3 Enviando Pull Requests (PR)
1. Crie um fork do repositório e ramifique a partir da branch `main`.
   * Nomenclatura sugerida: `feature/sua-feature` ou `bugfix/numero-da-issue`.
2. Garanta que o código esteja formatado segundo o padrão oficial do Go:
   ```bash
   go fmt ./...
   ```
3. Execute e garanta que toda a suíte de testes passe com sucesso:
   ```bash
   make test
   ```
4. Se você adicionou novas funcionalidades, adicione os respectivos testes unitários em `*_test.go` utilizando mocks de repositório.
5. Mantenha a documentação sincronizada: se você alterou parâmetros de banco, flags ou rotas REST, atualize os documentos correspondentes em `docs/`.

---

## 3. Diretrizes de Engenharia e Arquitetura

Para manter a consistência da base de código, exigimos o cumprimento rigoroso dos seguintes padrões:

* **Zero Variáveis Globais**: Dependências devem sempre ser passadas através dos construtores (`New...`) na raiz de composição ([`cmd/server/main.go`](cmd/server/main.go)).
* **Dependência em Abstrações**: O núcleo do sistema (`internal/monitor`, `internal/security`) deve depender apenas de interfaces declaradas em [`internal/domain`](internal/domain), nunca de implementações concretas de banco de dados ou rede.
* **Compatibilidade Dual-Engine**: Qualquer nova query SQL deve utilizar o [`storage.AdaptQuery`](docs/DATABASE.md) para garantir que funcione de forma transparente tanto em SQLite quanto em PostgreSQL.
* **Isolamento de Segurança**: Hashes de senha e tokens nunca devem ser expostos em JSON ou logs abertos.

Consulte o **[Guia Avançado do Desenvolvedor](docs/DEVELOPER_GUIDES.md)** para instruções detalhadas de desenvolvimento e arquitetura.

---

### 🔗 Documentos Relacionados
* 🏗️ [Arquitetura Geral](ARCHITECTURE.md) — Macro-estrutura do sistema
* 👨‍💻 [Guia do Desenvolvedor](docs/DEVELOPER_GUIDES.md) — Configuração do ambiente local
* 🧪 [Guia de Testes](docs/TESTING.md) — Execução de testes unitários e manuais
* 🔐 [Segurança e RBAC](docs/SECURITY.md) — Padrões de autenticação e proteção de rotas
* 🧭 [Central de Documentação](docs/INDEX.md) — Mapa de conteúdo completo
