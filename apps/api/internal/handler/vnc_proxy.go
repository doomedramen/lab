package handler

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/doomedramen/lab/apps/api/internal/service"
)

// VNCProxyHandler creates an HTTP handler that proxies WebSocket connections
// to a local VNC server. It validates a one-time token from the query string,
// looks up the VNC port, then upgrades to WebSocket and bidirectionally copies
// data between the browser and the libvirt VNC server.
func VNCProxyHandler(vmSvc *service.VMService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		ct, ok := vmSvc.ValidateConsoleToken(token)
		if !ok {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Connect to the local VNC server before upgrading WebSocket,
		// so we can return an error to the browser if the VNC port is unreachable.
		vncAddr := fmt.Sprintf("127.0.0.1:%d", ct.Port)
		vncConn, err := net.Dial("tcp", vncAddr)
		if err != nil {
			log.Printf("VNC proxy: failed to connect to %s: %v", vncAddr, err)
			http.Error(w, "VNC server unavailable", http.StatusBadGateway)
			return
		}

		// Upgrade the HTTP request to a WebSocket connection.
		// noVNC requires the "binary" subprotocol.
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{"binary"},
			InsecureSkipVerify: true, // CORS origin check is handled by our own middleware
		})
		if err != nil {
			vncConn.Close()
			log.Printf("VNC proxy: WebSocket upgrade failed: %v", err)
			return
		}

		netConn := websocket.NetConn(r.Context(), wsConn, websocket.MessageBinary)

		// Bidirectional proxy.
		done := make(chan struct{}, 2)
		go func() {
			if _, err := io.Copy(netConn, vncConn); err != nil {
				log.Printf("VNC proxy: VNC→WS copy error: %v", err)
			}
			done <- struct{}{}
		}()
		go func() {
			if _, err := io.Copy(vncConn, netConn); err != nil {
				log.Printf("VNC proxy: WS→VNC copy error: %v", err)
			}
			done <- struct{}{}
		}()

		// Wait for either direction to finish then close both sides.
		<-done
		vncConn.Close()
		netConn.Close()
	}
}
