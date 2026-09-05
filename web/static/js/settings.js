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
// File: web/static/js/settings.js
// Author: Gabriel Moraes
// Date: 2026-09-04

document.addEventListener('DOMContentLoaded', function () {

    // --- 1. View Switching Logic (Connected vs Setup) ---
    const connectedCard = document.getElementById('connected-card');
    const setupCard = document.getElementById('setup-card');
    const btnChangeAccount = document.getElementById('btn-change-account');
    const btnCancelEdit = document.getElementById('btn-cancel-edit');
    const feedbackAlert = document.getElementById('feedback-alert-smtp') || document.getElementById('feedback-alert');

    if (btnChangeAccount) {
        btnChangeAccount.addEventListener('click', () => {
            if (confirm('Are you sure you want to reconfigure the alert system?')) {
                connectedCard?.classList.add('d-none');
                setupCard?.classList.remove('d-none');
            }
        });
    }

    if (btnCancelEdit) {
        btnCancelEdit.addEventListener('click', () => {
            setupCard?.classList.add('d-none');
            connectedCard?.classList.remove('d-none');
        });
    }

    // --- 2. Provider Wizard Logic (Gmail/Outlook Helpers) ---
    // These elements only exist when the setup card is visible.
    const providerRadios = document.querySelectorAll('input[name="provider_select"]');
    const techSection = document.getElementById('technical_section');
    const hostInput = document.getElementById('smtp_host');
    const portInput = document.getElementById('smtp_port');
    const passLabel = document.getElementById('pass_label');
    const guidePanel = document.getElementById('guide_panel');
    const guideText = document.getElementById('guide_text');
    const guideLink = document.getElementById('guide_link');

    const providers = {
        'gmail': {
            host: 'smtp.gmail.com', port: '587',
            link: 'https://myaccount.google.com/apppasswords',
            text: 'Google blocks normal passwords. You must generate an <strong>App Password</strong>.',
            label: 'Paste Google App Password'
        },
        'outlook': {
            host: 'smtp.office365.com', port: '587',
            link: 'https://account.live.com/proofs/AppPassword',
            text: 'Microsoft requires an <strong>App Password</strong> if 2FA is active.',
            label: 'Paste Outlook App Password'
        },
        'custom': { host: '', port: '', label: 'Password' }
    };

    function updateFormUI(provider) {
        if (!techSection || !guidePanel) return; // Setup card not in DOM
        if (provider === 'custom') {
            techSection.classList.remove('d-none');
            guidePanel.classList.add('d-none');
            if (passLabel) passLabel.textContent = providers['custom'].label;
        } else {
            techSection.classList.add('d-none');
            if (providers[provider]) {
                if (hostInput) hostInput.value = providers[provider].host;
                if (portInput) portInput.value = providers[provider].port;
                guidePanel.classList.remove('d-none');
                if (guideText) guideText.innerHTML = providers[provider].text;
                if (guideLink) guideLink.href = providers[provider].link;
                if (passLabel) passLabel.textContent = providers[provider].label;
            }
        }
    }

    if (providerRadios.length > 0 && hostInput) {
        providerRadios.forEach(radio => {
            radio.addEventListener('change', (e) => updateFormUI(e.target.value));
        });

        // Detect initial state (when setup card visible with existing values)
        const currentHost = hostInput.value;
        if (currentHost === 'smtp.gmail.com') {
            const el = document.getElementById('prov_gmail');
            if (el) { el.checked = true; updateFormUI('gmail'); }
        } else if (currentHost === 'smtp.office365.com') {
            const el = document.getElementById('prov_outlook');
            if (el) { el.checked = true; updateFormUI('outlook'); }
        } else if (currentHost) {
            const el = document.getElementById('prov_custom');
            if (el) { el.checked = true; updateFormUI('custom'); }
        }
    }

    // --- 3. Test Email Connection ---
    const btnTest = document.getElementById('btn-test-existing');

    if (btnTest && feedbackAlert) {
        btnTest.addEventListener('click', function () {
            const originalHTML = btnTest.innerHTML;
            btnTest.disabled = true;
            btnTest.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Sending...';

            feedbackAlert.classList.add('d-none');
            feedbackAlert.classList.remove('alert-success', 'alert-danger');

            fetch('/settings/test', { method: 'POST' })
                .then(async response => {
                    const text = await response.text();
                    feedbackAlert.classList.remove('d-none');
                    if (response.ok) {
                        feedbackAlert.className = 'alert alert-success shadow-sm mb-4';
                        feedbackAlert.innerHTML = '<i class="fa-solid fa-check-circle me-2"></i><strong>Success!</strong> Test email sent correctly.';
                    } else {
                        feedbackAlert.className = 'alert alert-danger shadow-sm mb-4';
                        feedbackAlert.innerHTML = '<i class="fa-solid fa-triangle-exclamation me-2"></i><strong>Failed:</strong> ' + text;
                    }
                })
                .catch(error => {
                    feedbackAlert.classList.remove('d-none');
                    feedbackAlert.className = 'alert alert-danger shadow-sm mb-4';
                    feedbackAlert.innerHTML = 'Network Error: ' + error;
                })
                .finally(() => {
                    btnTest.disabled = false;
                    btnTest.innerHTML = originalHTML;
                });
        });
    }

    // --- 4. Telegram Bot Configuration ---
    const btnChangeTelegram = document.getElementById('btn-change-telegram');
    const btnCancelTelegram = document.getElementById('btn-cancel-telegram');
    const telegramConnected = document.getElementById('telegram-connected-card');
    const telegramSetup     = document.getElementById('telegram-setup-card');
    const telegramFeedback  = document.getElementById('feedback-alert-telegram');
    const btnTestTelegram   = document.getElementById('btn-test-telegram');

    function showTelegramAlert(msg, type) {
        if (!telegramFeedback) return;
        telegramFeedback.className = 'alert alert-' + type + ' shadow-sm mb-4';
        telegramFeedback.textContent = msg;
        telegramFeedback.classList.remove('d-none');
        setTimeout(() => telegramFeedback.classList.add('d-none'), 6000);
    }

    if (btnChangeTelegram) {
        btnChangeTelegram.addEventListener('click', () => {
            telegramConnected?.classList.add('d-none');
            telegramSetup?.classList.remove('d-none');
        });
    }

    if (btnCancelTelegram) {
        btnCancelTelegram.addEventListener('click', () => {
            telegramSetup?.classList.add('d-none');
            telegramConnected?.classList.remove('d-none');
        });
    }

    if (btnTestTelegram) {
        btnTestTelegram.addEventListener('click', async () => {
            const chatID = prompt('Enter your Telegram Chat ID to receive the test message:\n(Message @userinfobot on Telegram to get yours)');
            if (!chatID || !chatID.trim()) return;

            btnTestTelegram.disabled = true;
            btnTestTelegram.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i>Sending...';

            try {
                const fd = new FormData();
                fd.append('chat_id', chatID.trim());
                const res = await fetch('/settings/test-telegram', { method: 'POST', body: fd });
                const text = await res.text();
                if (res.ok) {
                    showTelegramAlert('✅ ' + text, 'success');
                } else {
                    showTelegramAlert('❌ ' + text, 'danger');
                }
            } catch (e) {
                showTelegramAlert('❌ Network error: ' + e.message, 'danger');
            } finally {
                btnTestTelegram.disabled = false;
                btnTestTelegram.innerHTML = '<i class="fa-brands fa-telegram me-2"></i>Send Test Message';
            }
        });
    }
});