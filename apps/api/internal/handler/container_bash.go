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

// resizeMsg is the JSON message sent by the client to resize the terminal.
type resizeMsg struct {
	Width  uint16 `json:"Width"`
	Height uint16 `json:"Height"`
}

// ContainerBashHandler creates an HTTP handler that opens a PTY bash session
// inside a Docker container and proxies it over WebSocket.
func ContainerBashHandler(stackSvc *service.StackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		ct, ok := stackSvc.ValidateContainerToken(token)
		if !ok {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Try /bin/bash first, fall back to /bin/sh
		shell := "/bin/bash"
		probe := exec.Command("docker", "exec", ct.ContainerName, "/bin/bash", "-c", "exit")
		if probe.Run() != nil {
			shell = "/bin/sh"
		}

		cmd := exec.Command("docker", "exec", "-it", ct.ContainerName, shell)
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")

		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Printf("ContainerBash: failed to start PTY for %s: %v", ct.ContainerName, err)
			http.Error(w, "failed to open container shell", http.StatusInternalServerError)
			return
		}
		defer func() {
			ptmx.Close()
			_ = cmd.Wait()
		}()

		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{"binary"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ContainerBash: WebSocket upgrade failed: %v", err)
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
						log.Printf("ContainerBash: PTY read error: %v", err)
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
						log.Printf("ContainerBash: WS read error: %v", err)
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
					log.Printf("ContainerBash: PTY write error: %v", werr)
					done <- struct{}{}
					return
				}
			}
		}()

		<-done
	}
}
