// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/desktop/app.go
// Author: Gabriel Moraes
// Date: 2026-01-20

package desktop

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"noxfort-monitor-server/internal/tray"
)

// App manages the Wails desktop window lifecycle.
type App struct {
	handler        http.Handler
	onShutdown     func()
	ctx            context.Context
	desktopTokenMu sync.RWMutex
	desktopToken   string
}

// New creates a new Desktop App instance.
func New(handler http.Handler, onShutdown func()) *App {
	return &App{
		handler:    handler,
		onShutdown: onShutdown,
	}
}

func (a *App) getDesktopToken() string {
	a.desktopTokenMu.RLock()
	defer a.desktopTokenMu.RUnlock()
	return a.desktopToken
}

func (a *App) setDesktopToken(token string) {
	a.desktopTokenMu.Lock()
	defer a.desktopTokenMu.Unlock()
	a.desktopToken = token
}

// desktopResponseWriter intercepts response headers to track session cookies set by HTTP handlers
// in embedded webviews (WebKitGTK) where custom URI scheme requests do not persist cookies natively.
type desktopResponseWriter struct {
	http.ResponseWriter
	syncSession func(http.Header)
	synced      bool
	mu          sync.Mutex
}

func (w *desktopResponseWriter) sync() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.synced {
		w.synced = true
		w.syncSession(w.ResponseWriter.Header())
	}
}

func (w *desktopResponseWriter) WriteHeader(code int) {
	w.sync()
	w.ResponseWriter.WriteHeader(code)
}

func (w *desktopResponseWriter) Write(b []byte) (int, error) {
	w.sync()
	return w.ResponseWriter.Write(b)
}

// ToggleFullscreen toggles between fullscreen and normal window state.
func (a *App) ToggleFullscreen() bool {
	if a.ctx != nil {
		if wailsruntime.WindowIsFullscreen(a.ctx) {
			wailsruntime.WindowUnfullscreen(a.ctx)
			return false
		} else {
			wailsruntime.WindowFullscreen(a.ctx)
			return true
		}
	}
	return false
}

// ExitFullscreen exits fullscreen if the window is currently fullscreen.
func (a *App) ExitFullscreen() {
	if a.ctx != nil && wailsruntime.WindowIsFullscreen(a.ctx) {
		wailsruntime.WindowUnfullscreen(a.ctx)
	}
}

// BuildAssetHandler returns the http.Handler used by Wails AssetServer that maintains
// session continuity across requests in WebKitGTK where custom URI schemes do not manage cookies.
func (a *App) BuildAssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 0. Handle Desktop Window Control APIs (F11 Fullscreen / Escape)
		if r.URL.Path == "/api/window/toggle-fullscreen" {
			isFullscreen := a.ToggleFullscreen()
			log.Printf("[DESKTOP] Fullscreen alternado (Fullscreen=%v).", isFullscreen)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    true,
				"fullscreen": isFullscreen,
			})
			return
		}
		if r.URL.Path == "/api/window/exit-fullscreen" {
			a.ExitFullscreen()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
			})
			return
		}

		// 1. Inject active desktop session token into incoming requests if missing
		token := a.getDesktopToken()
		if token != "" {
			if c, err := r.Cookie("noxfort_session"); err != nil || c.Value == "" {
				r.AddCookie(&http.Cookie{
					Name:  "noxfort_session",
					Value: token,
				})
			}
		}

		// 2. Intercept response to detect login session creation or logout deletion
		rw := &desktopResponseWriter{
			ResponseWriter: w,
			syncSession: func(h http.Header) {
				resp := http.Response{Header: h}
				for _, cookie := range resp.Cookies() {
					if cookie.Name == "noxfort_session" {
						if cookie.Value == "" || cookie.MaxAge < 0 {
							log.Println("[DESKTOP-AUTH] Sessão encerrada no Desktop Webview.")
							a.setDesktopToken("")
						} else {
							log.Printf("[DESKTOP-AUTH] Sessão autenticada e sincronizada no Desktop Webview.")
							a.setDesktopToken(cookie.Value)
						}
					}
				}
			},
		}

		log.Printf("[DESKTOP-WEBVIEW] %s %s", r.Method, r.URL.Path)
		a.handler.ServeHTTP(rw, r)
		rw.sync()
	})
}

// RestoreWindow brings the Wails window to the front and unminimizes it.
func (a *App) RestoreWindow() {
	if a.ctx != nil {
		log.Println("[DESKTOP] Trazendo janela para o primeiro plano...")
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
		wailsruntime.Show(a.ctx)
	}
}

// Run starts the Wails native desktop application.
func (a *App) Run() error {
	appOptions := &options.App{
		Title:             "Noxfort Monitor™",
		Width:             1280,
		Height:            800,
		MinWidth:          1024,
		MinHeight:         600,
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "noxfort-monitor-desktop-instance",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				a.RestoreWindow()
			},
		},
		AssetServer: &assetserver.Options{
			Handler: a.BuildAssetHandler(),
		},
		Linux: &linux.Options{
			Icon:             tray.IconBytes(),
			ProgramName:      "noxfort-monitor",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
		OnStartup: func(ctx context.Context) {
			a.ctx = ctx
			log.Println("[DESKTOP] Noxfort Monitor Wails Desktop Window initialized.")

			// Register system tray with callbacks that control the Wails window
			tray.Register(
				func() {
					log.Println("[TRAY] Abrir Interface solicitada.")
					a.RestoreWindow()
				},
				func() {
					log.Println("[TRAY] Encerrar solicitado.")
					wailsruntime.Quit(ctx)
				},
			)
		},
		OnShutdown: func(ctx context.Context) {
			log.Println("[DESKTOP] Noxfort Monitor Wails Desktop shutting down...")
			tray.Quit()
			if a.onShutdown != nil {
				a.onShutdown()
			}
		},
	}

	return wails.Run(appOptions)
}
