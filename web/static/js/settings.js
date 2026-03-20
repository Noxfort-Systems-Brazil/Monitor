// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: web/static/js/settings.js
// Author: Gabriel Moraes

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
});