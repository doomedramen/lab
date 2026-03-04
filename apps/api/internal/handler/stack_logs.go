package handler

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"nhooyr.io/websocket"

	"github.com/doomedramen/lab/apps/api/internal/service"
)

// StackLogsHandler creates an HTTP handler that streams docker compose logs
// for a stack over a WebSocket connection.
func StackLogsHandler(stackSvc *service.StackService, stacksDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		lt, ok := stackSvc.ValidateStackLogsToken(token)
		if !ok {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		composePath := filepath.Join(stacksDir, lt.StackID, "docker-compose.yml")

		// Upgrade to WebSocket first
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("StackLogs: WebSocket upgrade failed: %v", err)
			return
		}
		defer wsConn.Close(websocket.StatusNormalClosure, "done")

		cmd := exec.CommandContext(r.Context(), "docker", "compose", "-f", composePath, "logs", "-f", "--tail=200")

		// Use a combined pipe for both stdout and stderr
		pr, pw := io.Pipe()
		cmd.Stdout = pw
		cmd.Stderr = pw

		if err := cmd.Start(); err != nil {
			pw.Close()
			pr.Close()
			log.Printf("StackLogs: failed to start docker compose logs: %v", err)
			return
		}
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
			pw.Close()
			pr.Close()
		}()

		// Close the write end when the command exits
		go func() {
			_ = cmd.Wait()
			pw.Close()
		}()

		netConn := websocket.NetConn(r.Context(), wsConn, websocket.MessageText)

		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				if _, werr := netConn.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				if err != io.EOF && err != io.ErrClosedPipe {
					log.Printf("StackLogs: read error: %v", err)
				}
				break
			}
		}
	}
}
