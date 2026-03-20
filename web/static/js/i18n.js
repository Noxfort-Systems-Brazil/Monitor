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
// File: web/static/js/i18n.js
// Author: Gabriel Moraes
// Date: 2026-01-19

/**
 * Noxfort Monitor - Client-Side Internationalization (i18n) Engine
 * * This script handles the loading of translation JSON files and 
 * applies them to HTML elements with the 'data-i18n' attribute.
 */

const I18N_STORAGE_KEY = 'noxfort_language';
const DEFAULT_LANG = 'pt';
const SUPPORTED_LANGS = ['en', 'pt', 'es', 'fr', 'ru', 'zh']; // Atualizado para incluir todas as opções do menu

class I18nManager {
    constructor() {
        this.currentLang = this.detectLanguage();
        this.translations = {};
    }

    /**
     * Detects user's preferred language from LocalStorage or Browser Settings.
     */
    detectLanguage() {
        // 1. Check if user already selected a language previously
        const storedLang = localStorage.getItem(I18N_STORAGE_KEY);
        if (storedLang && SUPPORTED_LANGS.includes(storedLang)) {
            return storedLang;
        }

        // 2. Check browser language (e.g., "pt-BR" -> "pt")
        const browserLang = navigator.language.split('-')[0];
        if (SUPPORTED_LANGS.includes(browserLang)) {
            return browserLang;
        }

        // 3. Fallback
        return DEFAULT_LANG;
    }

    /**
     * Sets the language, loads dictionary, and updates UI.
     * @param {string} lang - Language code ('en', 'pt')
     */
    async setLanguage(lang) {
        if (!SUPPORTED_LANGS.includes(lang)) {
            console.error(`[i18n] Language '${lang}' not supported.`);
            return;
        }

        this.currentLang = lang;
        localStorage.setItem(I18N_STORAGE_KEY, lang);

        // Visual indicator on buttons (if they exist)
        this.updateActiveButton();

        await this.loadTranslations();
        this.applyTranslations();

        // Update the flag/label in the navbar dropdown
        this.updateDropdownLabel();
    }

    /**
     * Fetches the JSON file for the current language.
     */
    async loadTranslations() {
        try {
            const response = await fetch(`/static/locales/${this.currentLang}.json`);
            if (!response.ok) throw new Error('Failed to load language file');
            this.translations = await response.json();
            console.log(`[i18n] Loaded dictionary: ${this.currentLang}`);
        } catch (error) {
            console.error('[i18n] Error loading translations:', error);
        }
    }

    /**
     * Scans the DOM for 'data-i18n' attributes and replaces content.
     */
    applyTranslations() {
        // 1. data-i18n → textContent (or value/placeholder for inputs)
        const elements = document.querySelectorAll('[data-i18n]');
        elements.forEach(el => {
            const key = el.getAttribute('data-i18n');
            const translation = this.translations[key];
            if (translation) {
                if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
                    if (el.getAttribute('placeholder')) {
                        el.setAttribute('placeholder', translation);
                    } else {
                        el.value = translation;
                    }
                } else if (el.tagName === 'OPTION') {
                    el.textContent = translation;
                } else {
                    el.textContent = translation;
                }
            } else {
                console.warn(`[i18n] Missing key: ${key}`);
            }
        });

        // 2. data-i18n-placeholder → placeholder attribute
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            const translation = this.translations[key];
            if (translation) el.setAttribute('placeholder', translation);
        });

        // 3. data-role → translate role names stored in English from the DB
        const roleKeyMap = {
            'system admin': 'contact_role_admin',
            'technician': 'contact_role_technician',
            'programmer': 'contact_role_programmer',
        };
        document.querySelectorAll('[data-role]').forEach(el => {
            const role = (el.getAttribute('data-role') || '').toLowerCase();
            const key = roleKeyMap[role];
            if (key && this.translations[key]) el.textContent = this.translations[key];
        });

        // 4. Expose window._t for JS-side translations (confirm dialogs, etc.)
        window._t = (key) => this.translations[key] || key;

        // 5. Update HTML lang attribute for accessibility
        document.documentElement.lang = this.currentLang;

        // 6. Format the dashboard timestamp using Intl (so day names are localized)
        this.formatLocalizedDates();
    }

    /**
     * Finds <time> elements with a datetime attribute and reformats them
     * using Intl.DateTimeFormat in the current locale.
     */
    formatLocalizedDates() {
        const el = document.getElementById('dash-now');
        if (!el) return;
        const iso = el.getAttribute('datetime');
        if (!iso) return;
        try {
            const date = new Date(iso);
            el.textContent = new Intl.DateTimeFormat(this.currentLang, {
                weekday: 'short',
                year: 'numeric',
                month: 'short',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false,
            }).format(date);
        } catch (e) {
            console.warn('[i18n] Date format error:', e);
        }
    }

    updateActiveButton() {
        document.querySelectorAll('.lang-btn').forEach(btn => {
            btn.classList.remove('active');
            if (btn.getAttribute('data-lang') === this.currentLang) {
                btn.classList.add('active');
            }
        });
    }

    updateDropdownLabel() {
        const label = document.getElementById('current-lang-label');
        if (label) {
            label.textContent = this.currentLang.toUpperCase();
        }
    }
}

// Initialize on Page Load
const i18n = new I18nManager();

// Expose global applyI18n for scripts that modify DOM dynamically
window.applyI18n = () => i18n.applyTranslations();

document.addEventListener('DOMContentLoaded', async () => {
    await i18n.setLanguage(i18n.currentLang);

    // FIX: Changed selector from '.lang-switcher .lang-btn' to just '.lang-btn'
    // This ensures buttons work regardless of container class
    document.querySelectorAll('.lang-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            // Ensure we get the button even if clicking on an icon inside it (future proofing)
            const targetBtn = e.target.closest('.lang-btn');
            if (targetBtn) {
                const lang = targetBtn.getAttribute('data-lang');
                i18n.setLanguage(lang);
            }
        });
    });
});