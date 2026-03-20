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
// File: internal/transport/http/contact_handler.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package http

import (
	"html/template"
	"log"
	"net/http"
	"noxfort-monitor-server/internal/appdir"
	"strconv"

	"noxfort-monitor-server/internal/domain"
)

// ContactHandler manages the response team list.
type ContactHandler struct {
	repo domain.ContactRepository
}

// NewContactHandler creates the handler with dependencies.
func NewContactHandler(r domain.ContactRepository) *ContactHandler {
	return &ContactHandler{repo: r}
}

// ServePage renders the contacts management page.
func (h *ContactHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch contacts from DB
	contacts, err := h.repo.GetAllContacts()
	if err != nil {
		log.Printf("[CONTACTS] Failed to fetch contacts: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	// 2. Prepare Data
	data := map[string]interface{}{
		"Title":    "Response Team",
		"Contacts": contacts,
	}

	// 3. Render Template (Layout + Contacts)
	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/contacts.html"),
	)
	if err != nil {
		log.Printf("[CONTACTS] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	// Executes the layout, which will inject the "content" block defined in contacts.html
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[CONTACTS] Render error: %v", err)
	}
}

// HandleCreate adds a new contact.
func (h *ContactHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	contact := &domain.Contact{
		Name:           r.FormValue("name"),
		Email:          r.FormValue("email"),
		Phone:          r.FormValue("phone"),
		Role:           r.FormValue("role"),
		NotifyCritical: r.FormValue("notify_critical") == "on",
		Enabled:        true,
		TelegramChatID: r.FormValue("telegram_chat_id"),
	}

	if err := h.repo.CreateContact(contact); err != nil {
		log.Printf("[CONTACTS] Failed to create: %v", err)
		http.Error(w, "Failed to create contact", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

// HandleDelete removes a contact.
func (h *ContactHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Convert string ID to int
	idInt, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// FIX: Explicitly cast int to int64 required by the repository
	id := int64(idInt)

	if err := h.repo.DeleteContact(id); err != nil {
		log.Printf("[CONTACTS] Failed to delete: %v", err)
		http.Error(w, "Failed to delete contact", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}
