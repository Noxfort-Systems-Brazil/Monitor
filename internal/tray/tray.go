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
// File: internal/tray/tray.go
// Author: Gabriel Moraes
// Date: 2026-01-20

package tray

import (
	_ "embed"
	"log"

	"github.com/getlantern/systray"
)

//go:embed icon.png
var iconBytes []byte

// IconBytes returns the raw bytes of the tray icon.
func IconBytes() []byte {
	return iconBytes
}

// Register initializes the system tray inside an existing event loop (e.g. Wails / GTK).
func Register(onOpen func(), onExit func()) {
	systray.Register(func() {
		systray.SetIcon(iconBytes)
		systray.SetTooltip("Noxfort Monitor™")

		mOpen := systray.AddMenuItem("Abrir Interface", "Abrir a janela do aplicativo")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Encerrar / Sair", "Fechar o aplicativo completamente")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					log.Println("[TRAY] Abrir Interface clicado.")
					if onOpen != nil {
						onOpen()
					}
				case <-mQuit.ClickedCh:
					log.Println("[TRAY] Encerrar clicado no menu da bandeja.")
					if onExit != nil {
						onExit()
					} else {
						systray.Quit()
					}
				}
			}
		}()
	}, func() {
		log.Println("[TRAY] Tray closed.")
	})
}

// Start runs the system tray. It must be called on the main goroutine (OS requirement).
// onExit is called when the user clicks "Exit" from the tray menu.
func Start(onOpen func(), onExit func()) {
	systray.Run(func() {
		systray.SetIcon(iconBytes)
		systray.SetTooltip("Noxfort Monitor™ — Industrial Orchestration System")

		mOpen := systray.AddMenuItem("Abrir Interface", "Abrir a janela do aplicativo")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Encerrar / Sair", "Parar o servidor")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					if onOpen != nil {
						onOpen()
					}
				case <-mQuit.ClickedCh:
					log.Println("[TRAY] Encerrar clicado no menu da bandeja.")
					if onExit != nil {
						onExit()
					} else {
						systray.Quit()
					}
				}
			}
		}()
	}, onExit)
}

// Quit cleanly removes the tray icon.
func Quit() {
	systray.Quit()
}
