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
// File: web/static/js/remote.js
// Author: Gabriel Moraes
// Date: 2026-09-04

document.addEventListener('DOMContentLoaded', () => {
    // 1. Alert Feedback Element
    const feedbackBox = document.getElementById('remote-feedback');
    function showFeedback(message, type = 'success') {
        if (!feedbackBox) return;
        feedbackBox.className = `alert alert-${type} shadow-sm mb-4`;
        feedbackBox.innerHTML = message;
        feedbackBox.classList.remove('d-none');
        setTimeout(() => feedbackBox.classList.add('d-none'), 6000);
    }

    // Check URL parameters for ?success=1
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('success') === '1') {
        showFeedback('✅ Configurações salvas com sucesso! O túnel está sendo gerenciado em segundo plano.', 'success');
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    // 2. Clipboard Copy Buttons
    document.querySelectorAll('.copy-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
            const targetId = btn.getAttribute('data-target');
            const targetEl = document.getElementById(targetId);
            if (!targetEl) return;

            let textToCopy = '';
            if (targetEl.tagName === 'INPUT' || targetEl.tagName === 'TEXTAREA') {
                textToCopy = targetEl.value;
            } else {
                textToCopy = targetEl.textContent || targetEl.innerText;
            }

            if (!textToCopy) return;

            try {
                await navigator.clipboard.writeText(textToCopy);
                const originalHTML = btn.innerHTML;
                btn.innerHTML = '<i class="fa-solid fa-check text-success me-1"></i> Copiado!';
                btn.classList.add('btn-success');
                btn.classList.remove('btn-outline-info', 'btn-outline-secondary', 'btn-outline-light');
                setTimeout(() => {
                    btn.innerHTML = originalHTML;
                    btn.className = btn.className.replace('btn-success', '');
                    if (targetId === 'public-telemetry-url') {
                        btn.classList.add('btn-outline-info');
                    } else if (targetId === 'local-telemetry-url') {
                        btn.classList.add('btn-outline-secondary');
                    } else {
                        btn.classList.add('btn-outline-light');
                    }
                }, 2000);
            } catch (err) {
                console.error('Failed to copy: ', err);
                showFeedback('Falha ao copiar para a área de transferência', 'danger');
            }
        });
    });

    // 3. Toggle Visibility of Token Input
    const btnToggleToken = document.getElementById('btn-toggle-token');
    const tokenInput = document.getElementById('ngrok_auth_token');
    const iconToggleToken = document.getElementById('icon-toggle-token');
    if (btnToggleToken && tokenInput && iconToggleToken) {
        btnToggleToken.addEventListener('click', () => {
            if (tokenInput.type === 'password') {
                tokenInput.type = 'text';
                iconToggleToken.className = 'fa-solid fa-eye-slash text-warning';
            } else {
                tokenInput.type = 'password';
                iconToggleToken.className = 'fa-solid fa-eye';
            }
        });
    }

    // 4. Live Tunnel State Poller & Updater
    const badgeTunnelState = document.getElementById('badge-tunnel-state');
    const badgeTunnelStateText = document.getElementById('badge-tunnel-state-text');
    const publicUrlInput = document.getElementById('public-telemetry-url');
    const errorBox = document.getElementById('tunnel-error-box');
    const errorMsg = document.getElementById('tunnel-error-msg');
    const btnStartTunnel = document.getElementById('btn-start-tunnel');
    const btnStopTunnel = document.getElementById('btn-stop-tunnel');
    const startedTimeSpan = document.getElementById('tunnel-started-time');
    const dynamicCodeUrls = document.querySelectorAll('.dynamic-telemetry-url');

    let currentTunnelState = 'OFFLINE';
    let savedTokenValue = tokenInput ? tokenInput.value : '';
    let savedDomainValue = document.getElementById('ngrok_domain') ? document.getElementById('ngrok_domain').value : '';

    function updateTunnelUI(status) {
        if (!status) return;
        currentTunnelState = status.state;

        // Badge update
        if (badgeTunnelState && badgeTunnelStateText) {
            badgeTunnelState.className = 'badge rounded-pill px-3 py-2 fs-6';
            badgeTunnelStateText.textContent = status.state;

            if (status.state === 'ONLINE') {
                badgeTunnelState.classList.add('bg-success');
                badgeTunnelState.innerHTML = '<i class="fa-solid fa-circle-check me-1"></i> ONLINE';
            } else if (status.state === 'CONNECTING') {
                badgeTunnelState.classList.add('bg-warning', 'text-dark');
                badgeTunnelState.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> CONECTANDO';
            } else if (status.state === 'ERROR') {
                badgeTunnelState.classList.add('bg-danger');
                badgeTunnelState.innerHTML = '<i class="fa-solid fa-circle-exclamation me-1"></i> ERRO';
            } else {
                badgeTunnelState.classList.add('bg-secondary');
                badgeTunnelState.innerHTML = '<i class="fa-solid fa-circle-pause me-1"></i> OFFLINE';
            }
        }

        // Public URL update
        if (status.telemetry_url) {
            if (publicUrlInput) publicUrlInput.value = status.telemetry_url;
            dynamicCodeUrls.forEach(el => el.textContent = status.telemetry_url);
        } else if (status.domain) {
            const fallback = `https://${status.domain}/api/telemetry`;
            if (publicUrlInput) publicUrlInput.value = fallback;
        } else if (publicUrlInput && !publicUrlInput.value.startsWith('http')) {
            publicUrlInput.value = 'Aguardando conexão do túnel...';
        }

        // Error message box
        if (errorBox && errorMsg) {
            if (status.error_message) {
                errorMsg.textContent = status.error_message;
                errorBox.classList.remove('d-none');
            } else {
                errorBox.classList.add('d-none');
            }
        }

        // Action Buttons: Only ONE button visible at a time
        if (btnStartTunnel && btnStopTunnel) {
            if (status.state === 'ONLINE') {
                btnStartTunnel.classList.add('d-none');
                btnStopTunnel.classList.remove('d-none');
                btnStopTunnel.disabled = false;
                btnStopTunnel.innerHTML = '<i class="fa-solid fa-stop me-2"></i> Desconectar Túnel';
            } else if (status.state === 'CONNECTING') {
                btnStartTunnel.classList.remove('d-none');
                btnStartTunnel.disabled = true;
                btnStartTunnel.className = 'btn btn-warning text-dark py-2 fw-bold';
                btnStartTunnel.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Conectando Túnel...';
                btnStopTunnel.classList.add('d-none');
            } else {
                // OFFLINE or ERROR
                btnStartTunnel.classList.remove('d-none');
                btnStartTunnel.disabled = false;
                btnStartTunnel.className = 'btn btn-success py-2 fw-bold';
                btnStartTunnel.innerHTML = '<i class="fa-solid fa-play me-2"></i> Salvar e Iniciar Túnel';
                btnStopTunnel.classList.add('d-none');
            }
        }

        // Online success banner toggle
        const onlineBanner = document.getElementById('tunnel-online-banner');
        if (onlineBanner) {
            if (status.state === 'ONLINE') {
                onlineBanner.classList.remove('d-none');
            } else {
                onlineBanner.classList.add('d-none');
            }
        }

        if (startedTimeSpan && status.started_at) {
            startedTimeSpan.textContent = status.started_at;
        }
    }

    async function checkStatus() {
        try {
            const res = await fetch('/api/tunnel/status');
            if (res.ok) {
                const data = await res.json();
                updateTunnelUI(data);
                return data;
            }
        } catch (e) {
            console.error('Error fetching tunnel status:', e);
        }
        return null;
    }

    // Refresh Button
    const btnRefresh = document.getElementById('btn-refresh-status');
    if (btnRefresh) {
        btnRefresh.addEventListener('click', async () => {
            btnRefresh.disabled = true;
            btnRefresh.innerHTML = '<i class="fa-solid fa-arrows-rotate fa-spin me-1"></i> Verificando...';
            await checkStatus();
            setTimeout(() => {
                btnRefresh.disabled = false;
                btnRefresh.innerHTML = '<i class="fa-solid fa-arrows-rotate me-1"></i> Atualizar';
            }, 800);
        });
    }

    // Form Submit: Save & Start Tunnel (Single Unified Action)
    const tunnelForm = document.getElementById('tunnelConfigForm');
    if (tunnelForm) {
        tunnelForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            if (!btnStartTunnel) return;

            const tokenVal = tokenInput ? tokenInput.value.trim() : '';
            if (!tokenVal) {
                showFeedback('⚠️ Por favor, insira o seu <strong>Ngrok Authtoken</strong> antes de iniciar.', 'warning');
                if (tokenInput) tokenInput.focus();
                return;
            }

            btnStartTunnel.disabled = true;
            btnStartTunnel.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Salvando e Conectando...';

            try {
                const formData = new FormData(tunnelForm);
                const res = await fetch('/api/tunnel/save', {
                    method: 'POST',
                    body: formData,
                    headers: { 'Accept': 'application/json' }
                });

                if (res.ok) {
                    showFeedback('Configurações salvas com sucesso! Conectando túnel...', 'info');
                    // Update initial reference values
                    savedTokenValue = tokenVal;
                    const domInput = document.getElementById('ngrok_domain');
                    if (domInput) savedDomainValue = domInput.value.trim();

                    // Instantly reflect CONNECTING state in UI
                    updateTunnelUI({
                        state: 'CONNECTING',
                        domain: savedDomainValue,
                        telemetry_url: ''
                    });

                    pollStatusUntilOnline();
                } else {
                    const text = await res.text();
                    showFeedback(`Falha ao salvar configurações: ${text}`, 'danger');
                    btnStartTunnel.disabled = false;
                    btnStartTunnel.innerHTML = '<i class="fa-solid fa-play me-2"></i> Salvar e Iniciar Túnel';
                }
            } catch (err) {
                showFeedback('Erro de comunicação ao salvar: ' + err.message, 'danger');
                btnStartTunnel.disabled = false;
                btnStartTunnel.innerHTML = '<i class="fa-solid fa-play me-2"></i> Salvar e Iniciar Túnel';
            }
        });
    }

    // Disconnect Button
    if (btnStopTunnel) {
        btnStopTunnel.addEventListener('click', async () => {
            btnStopTunnel.disabled = true;
            btnStopTunnel.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Desconectando...';
            try {
                const res = await fetch('/api/tunnel/stop', { method: 'POST' });
                if (res.ok) {
                    showFeedback('Túnel desconectado com sucesso.', 'secondary');
                    await checkStatus();
                } else {
                    showFeedback('Erro ao desconectar túnel.', 'danger');
                    btnStopTunnel.disabled = false;
                    btnStopTunnel.innerHTML = '<i class="fa-solid fa-stop me-2"></i> Desconectar Túnel';
                }
            } catch (err) {
                showFeedback('Erro de rede ao desconectar: ' + err.message, 'danger');
                btnStopTunnel.disabled = false;
                btnStopTunnel.innerHTML = '<i class="fa-solid fa-stop me-2"></i> Desconectar Túnel';
            }
        });
    }

    // Detect field modifications while ONLINE to allow "Salvar e Reconectar"
    function checkDirtyFields() {
        if (currentTunnelState !== 'ONLINE' || !btnStartTunnel) return;
        const currentToken = tokenInput ? tokenInput.value.trim() : '';
        const currentDom = document.getElementById('ngrok_domain') ? document.getElementById('ngrok_domain').value.trim() : '';

        if (currentToken !== savedTokenValue || currentDom !== savedDomainValue) {
            btnStartTunnel.classList.remove('d-none');
            btnStartTunnel.disabled = false;
            btnStartTunnel.className = 'btn btn-warning text-dark py-2 fw-bold';
            btnStartTunnel.innerHTML = '<i class="fa-solid fa-rotate me-2"></i> Salvar e Reconectar';
        } else {
            btnStartTunnel.classList.add('d-none');
        }
    }

    if (tokenInput) tokenInput.addEventListener('input', checkDirtyFields);
    const domainField = document.getElementById('ngrok_domain');
    if (domainField) domainField.addEventListener('input', checkDirtyFields);

    // Auto-poll status periodically if connecting
    async function pollStatusUntilOnline() {
        let attempts = 0;
        const interval = setInterval(async () => {
            attempts++;
            const status = await checkStatus();
            if (status) {
                if (status.state === 'ONLINE') {
                    clearInterval(interval);
                    showFeedback('✅ Túnel conectado e operando com sucesso!', 'success');
                } else if (status.state === 'ERROR' || attempts >= 20) {
                    clearInterval(interval);
                    if (status.state === 'ERROR') {
                        showFeedback(`Erro ao conectar túnel: ${status.error_message || 'Verifique suas credenciais'}`, 'danger');
                    }
                }
            }
        }, 1200);
    }

    // 5. Initial status check on page load and automatic periodic polling
    checkStatus();
    const livePoller = setInterval(checkStatus, 3000);
    window.addEventListener('beforeunload', () => clearInterval(livePoller));
});
