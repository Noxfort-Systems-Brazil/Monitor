// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: web/static/js/settings_db_view.js
// Author: Gabriel Moraes
// Date: 2026-09-04
//
// Single Responsibility: Database DOM View & Presentation

class DatabaseView {
    constructor() {
        this.btnTestDB = document.getElementById('btn-test-db');
        this.btnSaveDB = document.getElementById('btn-save-db');
        this.btnRevertSQLite = document.getElementById('btn-revert-sqlite');
        this.revertContainer = document.getElementById('revert-sqlite-container');
        this.dbAlert = document.getElementById('feedback-alert-db');
        this.statusBadge = document.getElementById('db-status-badge');
        this.latencyDisplay = document.getElementById('db-latency-display');
        this.btnTogglePass = document.getElementById('btn-toggle-db-pass');
        this.passInput = document.getElementById('db_pass');
        this.eyeIcon = document.getElementById('eye-icon');
        this.modeNotice = document.getElementById('db-mode-notice');
        this.modeIcon = document.getElementById('db-mode-icon');
        this.modeText = document.getElementById('db-mode-text');

        // Provision modal elements
        this.btnOpenProvision = document.getElementById('btn-open-provision');
        this.modalProvisionEl = document.getElementById('modalProvisionPG');
        this.modalProvision = null;
        this.btnSubmitProvision = document.getElementById('btn-submit-provision');
        this.adminUserInput = document.getElementById('admin_user');
        this.adminPassInput = document.getElementById('admin_password');
        this.provAlert = document.getElementById('prov-alert');
        this.provTargetUser = document.getElementById('prov-target-user');
        this.provisionForm = document.getElementById('provisionUserForm');
        this.alertTimeout = null;
    }

    getFormData() {
        return {
            type: 'postgres',
            host: document.getElementById('db_host')?.value.trim() || 'localhost',
            port: parseInt(document.getElementById('db_port')?.value) || 5432,
            dbname: document.getElementById('db_name')?.value.trim() || 'banco_de_dados_noxfort',
            schema: document.getElementById('db_schema')?.value.trim() || 'schema_monitor',
            user: document.getElementById('db_user')?.value.trim() || 'user_synapse',
            password: document.getElementById('db_pass')?.value || '',
            sslmode: document.getElementById('db_sslmode')?.value || 'disable',
            migrate: document.getElementById('db_migrate')?.checked || false
        };
    }

    setFormData(config) {
        if (!config) return;
        if (config.type === 'postgres') {
            if (config.host && document.getElementById('db_host')) document.getElementById('db_host').value = config.host;
            if (config.port && document.getElementById('db_port')) document.getElementById('db_port').value = config.port;
            if (config.dbname && document.getElementById('db_name')) document.getElementById('db_name').value = config.dbname;
            if (config.schema && document.getElementById('db_schema')) document.getElementById('db_schema').value = config.schema;
            if (config.user && document.getElementById('db_user')) document.getElementById('db_user').value = config.user;
            if (config.sslmode && document.getElementById('db_sslmode')) document.getElementById('db_sslmode').value = config.sslmode;
        }
    }

    togglePasswordVisibility() {
        if (!this.passInput || !this.eyeIcon) return;
        if (this.passInput.type === 'password') {
            this.passInput.type = 'text';
            this.eyeIcon.classList.replace('fa-eye', 'fa-eye-slash');
        } else {
            this.passInput.type = 'password';
            this.eyeIcon.classList.replace('fa-eye-slash', 'fa-eye');
        }
    }

    renderStatus(status) {
        if (!status) return;
        if (status.connected && status.type === 'postgres') {
            if (this.statusBadge) {
                this.statusBadge.className = 'badge bg-success';
                this.statusBadge.innerHTML = `<i class="fa-solid fa-server me-1"></i> PostgreSQL (${status.dbname}.${status.schema})`;
            }
            if (this.modeNotice) {
                this.modeNotice.className = 'alert alert-success border-success d-flex align-items-center mb-4 py-2 px-3 bg-opacity-10';
            }
            if (this.modeIcon) {
                this.modeIcon.className = 'fa-solid fa-circle-check text-success me-2 fs-5';
            }
            if (this.modeText) {
                this.modeText.innerHTML = `Conectado ao <strong>PostgreSQL</strong> no schema <code>${status.schema}</code> do banco <code>${status.dbname}</code>. Auditoria industrial e sincronização ativas.`;
            }
            if (this.revertContainer) {
                this.revertContainer.classList.remove('d-none');
            }
            if (this.latencyDisplay) {
                this.latencyDisplay.textContent = `${status.latency_ms} ms (${status.version ? status.version.split(' ')[0] : 'OK'})`;
            }
        } else {
            if (this.statusBadge) {
                this.statusBadge.className = 'badge bg-warning text-dark';
                this.statusBadge.innerHTML = `<i class="fa-solid fa-file-shield me-1"></i> SQLite Local (Padrão)`;
            }
            if (this.modeNotice) {
                this.modeNotice.className = 'alert alert-dark border-secondary d-flex align-items-center mb-4 py-2 px-3';
            }
            if (this.modeIcon) {
                this.modeIcon.className = 'fa-solid fa-circle-info text-info me-2 fs-5';
            }
            if (this.modeText) {
                this.modeText.innerHTML = `O Monitor opera por padrão no <strong>SQLite local</strong> (zero configuração). Conecte ao PostgreSQL abaixo para compartilhar o ecossistema com CARINA e SYNAPSE.`;
            }
            if (this.revertContainer) {
                this.revertContainer.classList.add('d-none');
            }
            if (this.latencyDisplay) {
                this.latencyDisplay.textContent = '0 ms (SQLite Local)';
            }
        }
    }

    setSaveButtonState(state) {
        if (!this.btnSaveDB) return;
        if (state === 'connected') {
            this.btnSaveDB.className = 'btn btn-success flex-grow-1 py-2 fw-bold shadow-sm';
            this.btnSaveDB.innerHTML = '<i class="fa-solid fa-circle-check me-2"></i><span data-i18n="settings_db_connected_btn">Conectado</span>';
        } else if (state === 'saving') {
            this.btnSaveDB.className = 'btn btn-primary flex-grow-1 py-2 fw-bold';
            this.btnSaveDB.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span><span data-i18n="settings_db_saving_btn">Salvando e Conectando...</span>';
        } else {
            this.btnSaveDB.className = 'btn btn-primary flex-grow-1 py-2 fw-bold';
            this.btnSaveDB.innerHTML = '<i class="fa-solid fa-floppy-disk me-2"></i><span data-i18n="settings_db_save_btn">Salvar e Conectar</span>';
        }
        if (window.applyI18n) window.applyI18n();
    }

    setTestButtonState(state) {
        if (!this.btnTestDB) return;
        if (state === 'testing') {
            this.btnTestDB.disabled = true;
            this.btnTestDB.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span><span data-i18n="settings_db_testing_btn">Testando...</span>';
        } else if (state === 'ok') {
            this.btnTestDB.disabled = false;
            this.btnTestDB.className = 'btn btn-outline-success flex-grow-1 py-2';
            this.btnTestDB.innerHTML = '<i class="fa-solid fa-check me-2"></i><span data-i18n="settings_db_tested_ok">Conexão OK!</span>';
        } else if (state === 'fail') {
            this.btnTestDB.disabled = false;
            this.btnTestDB.className = 'btn btn-outline-danger flex-grow-1 py-2';
            this.btnTestDB.innerHTML = '<i class="fa-solid fa-triangle-exclamation me-2"></i><span data-i18n="settings_db_test_btn">Testar Conexão</span>';
        } else {
            this.btnTestDB.disabled = false;
            this.btnTestDB.className = 'btn btn-outline-primary flex-grow-1 py-2';
            this.btnTestDB.innerHTML = '<i class="fa-solid fa-plug me-2"></i><span data-i18n="settings_db_test_btn">Testar Conexão</span>';
        }
        if (window.applyI18n) window.applyI18n();
    }

    getModalProvision() {
        if (!this.modalProvision && this.modalProvisionEl && window.bootstrap && typeof window.bootstrap.Modal === 'function') {
            if (typeof window.bootstrap.Modal.getOrCreateInstance === 'function') {
                this.modalProvision = window.bootstrap.Modal.getOrCreateInstance(this.modalProvisionEl);
            } else {
                this.modalProvision = new window.bootstrap.Modal(this.modalProvisionEl);
            }
        }
        return this.modalProvision;
    }

    openProvisionModal(targetUser) {
        if (this.provTargetUser) {
            this.provTargetUser.textContent = targetUser || 'user_monitor';
        }
        if (this.provAlert) {
            this.provAlert.className = 'alert d-none small mb-3';
            this.provAlert.innerHTML = '';
        }
        if (this.adminPassInput) {
            this.adminPassInput.value = '';
        }
        const modal = this.getModalProvision();
        if (modal) {
            modal.show();
        }
    }

    closeProvisionModal() {
        const modal = this.getModalProvision();
        if (modal) {
            modal.hide();
        }
    }

    showProvisionAlert(msg, type) {
        if (!this.provAlert) return;
        this.provAlert.className = `alert alert-${type} small mb-3`;
        this.provAlert.innerHTML = msg;
        this.provAlert.classList.remove('d-none');
    }

    getAdminCredentials() {
        return {
            admin_user: this.adminUserInput?.value.trim() || 'postgres',
            admin_password: this.adminPassInput?.value || ''
        };
    }

    setProvisionButtonState(state) {
        if (!this.btnSubmitProvision) return;
        if (state === 'loading') {
            this.btnSubmitProvision.disabled = true;
            this.btnSubmitProvision.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Criando usuário...';
        } else {
            this.btnSubmitProvision.disabled = false;
            this.btnSubmitProvision.innerHTML = '<i class="fa-solid fa-wand-magic-sparkles me-2"></i> Criar Usuário e Conectar';
        }
    }

    showAlert(msg, type, onQuickAction) {
        if (!this.dbAlert) return;
        this.dbAlert.className = `alert alert-${type} shadow-sm mb-4`;
        this.dbAlert.innerHTML = msg;
        this.dbAlert.classList.remove('d-none');

        if (onQuickAction) {
            const btn = this.dbAlert.querySelector('#btn-quick-provision');
            if (btn) {
                btn.addEventListener('click', (e) => {
                    e.preventDefault();
                    onQuickAction();
                });
            }
        }

        if (this.alertTimeout) {
            clearTimeout(this.alertTimeout);
        }
        this.alertTimeout = setTimeout(() => this.dbAlert?.classList.add('d-none'), 12000);
    }

    bindEvents({ onTogglePass, onFormChange, onTest, onSave, onRevertSQLite, onOpenProvision, onSubmitProvision }) {
        this.btnTogglePass?.addEventListener('click', onTogglePass);

        const dbFormInputs = document.querySelectorAll('#dbSettingsForm input, #dbSettingsForm select');
        dbFormInputs.forEach(el => {
            el.addEventListener('input', onFormChange);
            el.addEventListener('change', onFormChange);
        });

        this.btnTestDB?.addEventListener('click', onTest);
        this.btnSaveDB?.addEventListener('click', onSave);
        this.btnRevertSQLite?.addEventListener('click', onRevertSQLite);
        this.btnOpenProvision?.addEventListener('click', onOpenProvision);
        this.btnSubmitProvision?.addEventListener('click', onSubmitProvision);

        this.provisionForm?.addEventListener('submit', (e) => {
            e.preventDefault();
            if (onSubmitProvision) onSubmitProvision();
        });
    }
}
