// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: web/static/js/users.js
// Author: Gabriel Moraes
// Date: 2026-09-04

document.addEventListener('DOMContentLoaded', () => {
    // 1. SuperUser Authentication Form
    const loginForm = document.getElementById('loginForm');
    const loginFeedback = document.getElementById('login-feedback');
    const btnLoginSubmit = document.getElementById('btn-login-submit');

    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('auth_username')?.value.trim();
            const password = document.getElementById('auth_password')?.value;

            if (!username || !password) return;

            if (btnLoginSubmit) {
                btnLoginSubmit.disabled = true;
                btnLoginSubmit.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i>Autenticando...';
            }

            try {
                const fd = new FormData();
                fd.append('username', username);
                fd.append('password', password);

                const res = await fetch('/api/auth/login', {
                    method: 'POST',
                    body: fd
                });

                const data = await res.json();

                if (res.ok && data.success) {
                    if (loginFeedback) {
                        loginFeedback.className = 'alert alert-success shadow-sm mb-3';
                        loginFeedback.textContent = '✅ Autenticação realizada com sucesso. Carregando...';
                        loginFeedback.classList.remove('d-none');
                    }
                    setTimeout(() => {
                        window.location.reload();
                    }, 500);
                } else {
                    let msg = data.message || 'Credenciais inválidas.';
                    if (data.is_lockdown) {
                        msg = '🚨 SISTEMA EM LOCKDOWN! Apenas o Super Usuário Mestre pode desbloquear.';
                    } else if (data.failed_attempts > 0) {
                        msg += ` (Tentativas: ${data.failed_attempts}/3)`;
                    }

                    if (loginFeedback) {
                        loginFeedback.className = 'alert alert-danger shadow-sm mb-3';
                        loginFeedback.textContent = '❌ ' + msg;
                        loginFeedback.classList.remove('d-none');
                    }
                    if (btnLoginSubmit) {
                        btnLoginSubmit.disabled = false;
                        btnLoginSubmit.innerHTML = '<i class="fa-solid fa-unlock me-2"></i>Autenticar';
                    }
                }
            } catch (err) {
                if (loginFeedback) {
                    loginFeedback.className = 'alert alert-danger shadow-sm mb-3';
                    loginFeedback.textContent = '❌ Falha de comunicação: ' + err.message;
                    loginFeedback.classList.remove('d-none');
                }
                if (btnLoginSubmit) {
                    btnLoginSubmit.disabled = false;
                    btnLoginSubmit.innerHTML = '<i class="fa-solid fa-unlock me-2"></i>Autenticar';
                }
            }
        });
    }

    // 2. Lock / End Session
    const btnLogout = document.getElementById('btn-logout');
    if (btnLogout) {
        btnLogout.addEventListener('click', async () => {
            try {
                try { localStorage.removeItem('noxfort_session'); } catch(e){}
                await fetch('/api/auth/logout', {
                    method: 'POST',
                    headers: { 'Accept': 'application/json' }
                });
                window.location.reload();
            } catch (e) {
                window.location.href = '/users';
            }
        });
    }

    // 3. Add User Form
    const addUserForm = document.getElementById('addUserForm');
    const actionFeedback = document.getElementById('action-feedback');
    const btnAddUser = document.getElementById('btn-add-user');

    function showActionAlert(msg, type = 'success') {
        if (!actionFeedback) return;
        actionFeedback.className = `alert alert-${type} shadow-sm mb-4`;
        actionFeedback.textContent = msg;
        actionFeedback.classList.remove('d-none');
        setTimeout(() => actionFeedback.classList.add('d-none'), 5000);
    }

    if (addUserForm) {
        addUserForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('new_username')?.value.trim();
            const password = document.getElementById('new_password')?.value;
            const role = document.getElementById('new_role')?.value;

            if (!username || !password) return;

            if (btnAddUser) {
                btnAddUser.disabled = true;
                btnAddUser.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i>Cadastrando...';
            }

            try {
                const fd = new FormData();
                fd.append('username', username);
                fd.append('password', password);
                fd.append('role', role);

                const res = await fetch('/api/users/create', {
                    method: 'POST',
                    body: fd
                });

                const data = await res.json();
                if (res.ok && data.success) {
                    showActionAlert(`✅ Usuário "${username}" cadastrado com sucesso!`, 'success');
                    addUserForm.reset();
                    setTimeout(() => window.location.reload(), 1000);
                } else {
                    showActionAlert('❌ ' + (data.message || 'Erro ao cadastrar usuário'), 'danger');
                }
            } catch (err) {
                showActionAlert('❌ Erro de rede: ' + err.message, 'danger');
            } finally {
                if (btnAddUser) {
                    btnAddUser.disabled = false;
                    btnAddUser.innerHTML = '<i class="fa-solid fa-plus me-2"></i>Criar Usuário';
                }
            }
        });
    }

    // 4. Delete User Buttons
    document.querySelectorAll('.btn-delete-user').forEach(btn => {
        btn.addEventListener('click', async () => {
            const username = btn.getAttribute('data-username');
            if (!username) return;

            if (!confirm(`Tem certeza que deseja remover o usuário "${username}"?`)) {
                return;
            }

            try {
                const fd = new FormData();
                fd.append('username', username);

                const res = await fetch('/api/users/delete', {
                    method: 'POST',
                    body: fd
                });

                const data = await res.json();
                if (res.ok && data.success) {
                    showActionAlert(`✅ Usuário "${username}" removido.`, 'success');
                    const row = document.getElementById(`user-row-${username}`);
                    if (row) row.remove();
                } else {
                    showActionAlert('❌ ' + (data.message || 'Falha ao remover usuário.'), 'danger');
                }
            } catch (err) {
                showActionAlert('❌ Erro de rede: ' + err.message, 'danger');
            }
        });
    });
});
