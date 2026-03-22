package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"nhooyr.io/websocket"

	"github.com/doomedramen/lab/apps/api/internal/service"
)

// HostShellHandler creates an HTTP handler that opens a local PTY shell
// and proxies it over WebSocket. This allows terminal access to the host
// system directly from the web UI.
func HostShellHandler(nodeSvc *service.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		ht, ok := nodeSvc.ValidateHostShellToken(token)
		if !ok {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Verify the node exists and is accessible
		// For local host, we just verify the token was valid
		// In a distributed setup, this would route to the correct node
		_ = ht.NodeID // Used for logging/auditing in production

		// Determine the shell to use
		shell := determineShell()

		// Start the shell with a PTY
		cmd := exec.Command(shell)
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")
		// Set HOME to ensure shell config files are loaded
		if home := os.Getenv("HOME"); home != "" {
			cmd.Dir = home
		}

		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Printf("HostShell: failed to start PTY: %v", err)
			http.Error(w, "failed to open shell", http.StatusInternalServerError)
			return
		}
		defer func() {
			ptmx.Close()
			_ = cmd.Wait()
		}()

		// Upgrade to WebSocket
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{"binary"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("HostShell: WebSocket upgrade failed: %v", err)
			return
		}
		defer wsConn.Close(websocket.StatusNormalClosure, "done")

		netConn := websocket.NetConn(r.Context(), wsConn, websocket.MessageBinary)

		done := make(chan struct{}, 2)

		// PTY -> WebSocket
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if _, werr := netConn.Write(buf[:n]); werr != nil {
						done <- struct{}{}
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						log.Printf("HostShell: PTY read error: %v", err)
					}
					done <- struct{}{}
					return
				}
			}
		}()

		// WebSocket -> PTY (or resize message)
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := netConn.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("HostShell: WS read error: %v", err)
					}
					done <- struct{}{}
					return
				}
				if n == 0 {
					continue
				}

				// Attempt to parse as a resize message
				var resize resizeMsg
				if json.Unmarshal(buf[:n], &resize) == nil && resize.Width > 0 && resize.Height > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{Cols: resize.Width, Rows: resize.Height})
					continue
				}

				if _, werr := ptmx.Write(buf[:n]); werr != nil {
					log.Printf("HostShell: PTY write error: %v", werr)
					done <- struct{}{}
					return
				}
			}
		}()

		<-done
	}
}

// determineShell returns the preferred shell binary path
func determineShell() string {
	// Try user's preferred shell from SHELL env
	if shell := os.Getenv("SHELL"); shell != "" {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Fall back to common shells in order of preference
	shells := []string{"/bin/bash", "/usr/bin/bash", "/bin/zsh", "/usr/bin/zsh", "/bin/sh"}
	for _, shell := range shells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Ultimate fallback
	return "/bin/sh"
}
