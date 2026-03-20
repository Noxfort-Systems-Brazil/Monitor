// Noxfort Monitor™ - System Tray Integration
// File: internal/tray/tray.go

package tray

import (
	_ "embed"
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed icon.png
var iconBytes []byte

// Start runs the system tray. It must be called on the main goroutine (OS requirement).
// onExit is called when the user clicks "Exit" from the tray menu.
func Start(port string, onExit func()) {
	systray.Run(func() {
		onReady(port)
	}, onExit)
}

// Quit cleanly removes the tray icon.
func Quit() {
	systray.Quit()
}

func onReady(port string) {
	systray.SetIcon(iconBytes)
	systray.SetTooltip("Noxfort Monitor™ — Industrial Orchestration System")

	mOpen := systray.AddMenuItem("Open Interface", "Open the web dashboard")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit / Quit", "Stop the server")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				url := fmt.Sprintf("http://localhost:%s", port)
				openBrowser(url)
			case <-mQuit.ClickedCh:
				log.Println("[TRAY] Exit requested from system tray.")
				systray.Quit()
			}
		}
	}()
}

// openBrowser tries to open the URL in the system default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Printf("[TRAY] Failed to open browser: %v", err)
		}
	}
}
