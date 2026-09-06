[📚 Documentation Hub](docs/INDEX.md) > **Contributing Guide**

---

# 🤝 Contributing to Noxfort Monitor™

Thank you for your interest in contributing to **Noxfort Monitor™**. This project is maintained with high standards of architectural integrity, **SOLID** principles, and automated test coverage.

---

## 1. Code of Conduct

When participating in this project, all contributors are expected to treat the community with respect, maintain professional and constructive communication, and uphold code quality and security.

---

## 2. How Can I Contribute?

### 2.1 Reporting Bugs
* **Search Existing Issues**: Check whether the issue has already been reported.
* **Environment Details**: Always provide the Go version (`go version`), Linux distribution (`lsb_release -a`), Mosquitto version, and whether the bug occurs in Desktop mode (Wails) or Headless mode (`--headless`).
* **Relevant Logs**: Attach console logs or execute `journalctl -u noxfort-monitor -f`.

### 2.2 Suggesting Improvements & New Features
* Open an issue describing the industrial use case.
* If the improvement involves new database tables or drivers, refer to [Database & Persistence](docs/DATABASE.md).
* If the improvement introduces new communication endpoints, refer to [API Reference](docs/API_REFERENCE.md).

### 2.3 Submitting Pull Requests (PR)
1. Fork the repository and branch from `main`.
   * Suggested naming: `feature/your-feature-name` or `bugfix/issue-number`.
2. Ensure code is formatted according to official Go standards:
   ```bash
   go fmt ./...
   ```
3. Run and ensure the entire test suite passes:
   ```bash
   make test
   ```
4. If you added new features, add corresponding unit tests in `*_test.go` using repository mocks.
5. Keep documentation synchronized: if you changed database parameters, flags, or REST routes, update the relevant documents in `docs/`.

---

## 3. Engineering & Architectural Guidelines

To maintain codebase consistency, strict compliance with the following standards is required:

* **Zero Global Variables**: Dependencies must always be injected via constructors (`New...`) at the composition root ([`cmd/server/main.go`](cmd/server/main.go)).
* **Depend on Abstractions**: Core system logic (`internal/monitor`, `internal/security`) must only depend on interfaces declared in [`internal/domain`](internal/domain), never on concrete database or network implementations.
* **Dual-Engine Compatibility**: Any new SQL query must use [`storage.AdaptQuery`](docs/DATABASE.md) to ensure transparent operation across both SQLite and PostgreSQL.
* **Security Isolation**: Password hashes and tokens must never be exposed in JSON payloads or plaintext logs.

Refer to the **[Advanced Developer Guide](docs/DEVELOPER_GUIDES.md)** for detailed development and architectural instructions.

---

### 🔗 Related Documentation
* 🏗️ [System Architecture](ARCHITECTURE.md) — Macro-level system structure
* 👨‍💻 [Developer Guide](docs/DEVELOPER_GUIDES.md) — Local environment configuration
* 🧪 [Testing Guide](docs/TESTING.md) — Running unit and manual tests
* 🔐 [Security & RBAC](docs/SECURITY.md) — Authentication standards and route protection
* 🧭 [Documentation Hub](docs/INDEX.md) — Comprehensive content map
