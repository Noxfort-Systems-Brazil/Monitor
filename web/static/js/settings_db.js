// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: web/static/js/settings_db.js
// Author: Gabriel Moraes
// Date: 2026-09-04
//
// Single Responsibility: Database Settings Presenter & State Orchestration (DIP + SRP)

class DatabaseSettingsController {
    /**
     * @param {DatabaseApiClient} api
     * @param {DatabaseView} view
     */
    constructor(api, view) {
        this.api = api;
        this.view = view;
        this.lastConnectedConfigStr = '';
        this.isConnected = false;
    }

    init() {
        if (!this.view.btnSaveDB) return;

        this.view.bindEvents({
            onTogglePass: () => this.view.togglePasswordVisibility(),
            onFormChange: () => this.handleFormChange(),
            onTest: () => this.handleTest(),
            onSave: () => this.handleSave(),
            onRevertSQLite: () => this.handleRevertSQLite(),
            onOpenProvision: () => this.handleOpenProvision(),
            onSubmitProvision: () => this.handleProvisionUser()
        });

        this.loadStatus();
    }

    getRelevantConfigString(payload) {
        return JSON.stringify({
            host: payload.host,
            port: payload.port,
            dbname: payload.dbname,
            schema: payload.schema,
            user: payload.user,
            password: payload.password,
            sslmode: payload.sslmode
        });
    }

    handleFormChange() {
        if (!this.isConnected) {
            this.view.setSaveButtonState('save');
            return;
        }
        const currentRelevant = this.getRelevantConfigString(this.view.getFormData());
        if (currentRelevant !== this.lastConnectedConfigStr) {
            this.view.setSaveButtonState('save');
        } else {
            this.view.setSaveButtonState('connected');
        }
    }

    handleOpenProvision() {
        const formData = this.view.getFormData();
        const targetUser = formData.user || 'user_monitor';
        this.view.openProvisionModal(targetUser);
    }

    async handleProvisionUser() {
        const formData = this.view.getFormData();
        const adminCreds = this.view.getAdminCredentials();

        if (!formData.user) {
            this.view.showProvisionAlert('Informe o nome do usuário a ser criado no formulário.', 'warning');
            return;
        }
        if (!formData.password) {
            this.view.showProvisionAlert('Informe uma senha no campo Senha do formulário principal antes de provisionar.', 'warning');
            return;
        }
        if (!adminCreds.admin_password) {
            this.view.showProvisionAlert('Informe a senha do administrador do PostgreSQL.', 'warning');
            return;
        }

        this.view.setProvisionButtonState('loading');

        try {
            const payload = {
                host: formData.host,
                port: formData.port,
                dbname: formData.dbname,
                schema: formData.schema,
                sslmode: formData.sslmode,
                migrate: formData.migrate,
                admin_user: adminCreds.admin_user,
                admin_password: adminCreds.admin_password,
                new_user: formData.user,
                new_password: formData.password
            };

            const { ok, data } = await this.api.provisionUser(payload);
            if (ok && data.success) {
                this.view.closeProvisionModal();
                this.view.showAlert(`🎉 <strong>Sucesso!</strong> ${data.message}`, 'success');
                await this.loadStatus();
            } else {
                this.view.showProvisionAlert(`❌ ${data?.error || 'Erro ao provisionar usuário'}`, 'danger');
            }
        } catch (err) {
            this.view.showProvisionAlert(`❌ Erro de rede: ${err.message}`, 'danger');
        } finally {
            this.view.setProvisionButtonState('default');
        }
    }

    isAuthError(errorMsg) {
        if (!errorMsg) return false;
        const msg = errorMsg.toLowerCase();
        return msg.includes('28p01') ||
               msg.includes('password authentication failed') ||
               (msg.includes('role') && msg.includes('does not exist'));
    }

    displayErrorWithProvisionOption(errorMsg) {
        if (this.isAuthError(errorMsg)) {
            const user = this.view.getFormData().user || 'user_monitor';
            const alertMsg = `
                <div class="d-flex flex-column flex-md-row justify-content-between align-items-start align-items-md-center gap-2">
                    <div>
                        <strong>Falha de Autenticação:</strong> Usuário <code>${user}</code> não existe ou a senha está incorreta.
                    </div>
                    <button type="button" class="btn btn-warning btn-sm fw-bold text-nowrap" id="btn-quick-provision">
                        <i class="fa-solid fa-wand-magic-sparkles me-1"></i> Criar "${user}" no PostgreSQL agora
                    </button>
                </div>
            `;
            this.view.showAlert(alertMsg, 'danger', () => this.handleOpenProvision());
        } else {
            this.view.showAlert(`❌ <strong>Erro:</strong> ${errorMsg}`, 'danger');
        }
    }

    async loadStatus() {
        try {
            const data = await this.api.getStatus();
            if (!data) return;

            if (data.config) {
                this.view.setFormData(data.config);
            }

            if (data.status) {
                this.view.renderStatus(data.status);
                this.isConnected = (data.status.type === 'postgres' && Boolean(data.status.connected));

                if (this.isConnected) {
                    this.lastConnectedConfigStr = this.getRelevantConfigString(this.view.getFormData());
                    this.view.setSaveButtonState('connected');
                } else {
                    this.view.setSaveButtonState('save');
                }
            }
        } catch (e) {
            console.error('[DB Settings Controller] Failed to fetch database status:', e);
        }
    }

    async handleTest() {
        this.view.setTestButtonState('testing');

        try {
            const payload = this.view.getFormData();
            const { ok, data } = await this.api.testConnection(payload);

            if (ok && data.success) {
                this.view.showAlert(`✅ <strong>Conexão bem-sucedida!</strong> ${data.message} <span class="badge bg-secondary ms-2">${data.status.latency_ms} ms</span>`, 'success');
                this.view.setTestButtonState('ok');
                setTimeout(() => this.view.setTestButtonState('default'), 3000);
            } else {
                this.displayErrorWithProvisionOption(data?.error || 'Erro desconhecido');
                this.view.setTestButtonState('fail');
                setTimeout(() => this.view.setTestButtonState('default'), 3000);
            }
        } catch (e) {
            this.view.showAlert(`❌ <strong>Erro de rede:</strong> ${e.message}`, 'danger');
            this.view.setTestButtonState('default');
        }
    }

    async handleSave() {
        const payload = this.view.getFormData();
        const currentRelevant = this.getRelevantConfigString(payload);

        // If already connected and form hasn't changed, re-validate
        if (this.isConnected && currentRelevant === this.lastConnectedConfigStr) {
            this.view.setSaveButtonState('saving');
            await this.loadStatus();
            this.view.setSaveButtonState('connected');
            this.view.showAlert('✅ <strong>Banco de dados conectado e operacional!</strong>', 'success');
            return;
        }

        this.view.setSaveButtonState('saving');

        try {
            const { ok, data } = await this.api.saveConnection(payload);
            if (ok && data.success) {
                this.view.showAlert(`🎉 <strong>Sucesso!</strong> ${data.message}`, 'success');
                await this.loadStatus();
            } else {
                this.displayErrorWithProvisionOption(data?.error || 'Falha desconhecida');
                this.view.setSaveButtonState('save');
            }
        } catch (e) {
            this.view.showAlert(`❌ <strong>Erro de rede:</strong> ${e.message}`, 'danger');
            this.view.setSaveButtonState('save');
        }
    }

    async handleRevertSQLite() {
        if (!confirm('Deseja realmente desconectar do PostgreSQL e voltar a operar no SQLite local embutido?')) {
            return;
        }
        this.view.showAlert('Desconectando do PostgreSQL e reativando SQLite local...', 'info');
        try {
            const { ok, data } = await this.api.saveConnection({ type: 'sqlite', migrate: false });
            if (ok && data.success) {
                this.view.showAlert('✅ Operando no SQLite local com sucesso.', 'success');
                await this.loadStatus();
            } else {
                this.view.showAlert(`❌ Erro ao voltar para SQLite: ${data?.error || 'Erro desconhecido'}`, 'danger');
            }
        } catch (e) {
            this.view.showAlert(`❌ Falha de comunicação: ${e.message}`, 'danger');
        }
    }
}

// Composition Root (Dependency Injection)
document.addEventListener('DOMContentLoaded', () => {
    const api = new DatabaseApiClient();
    const view = new DatabaseView();
    const controller = new DatabaseSettingsController(api, view);
    controller.init();
});
