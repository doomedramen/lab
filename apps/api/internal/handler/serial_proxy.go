package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/doomedramen/lab/apps/api/internal/service"
	libvirt "libvirt.org/go/libvirt"
)

// SerialProxyHandler creates an HTTP handler that proxies WebSocket connections
// to a VM's serial console. It validates a one-time token, opens the serial
// console via libvirt, then upgrades to WebSocket and bidirectionally copies
// data between the browser and the serial console.
func SerialProxyHandler(vmSvc *service.VMService) http.HandlerFunc {
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

		// Get VM to find its domain name
		vm, err := vmSvc.GetByVMID(r.Context(), ct.VMID)
		if err != nil {
			log.Printf("Serial proxy: failed to get VM %d: %v", ct.VMID, err)
			http.Error(w, "VM not found", http.StatusNotFound)
			return
		}

		// Connect to libvirt
		conn, err := libvirt.NewConnect("qemu:///system")
		if err != nil {
			log.Printf("Serial proxy: failed to connect to libvirt: %v", err)
			http.Error(w, "Failed to connect to hypervisor", http.StatusBadGateway)
			return
		}
		defer conn.Close()

		// Lookup the domain by name
		domain, err := conn.LookupDomainByName(vm.ID)
		if err != nil {
			log.Printf("Serial proxy: failed to lookup domain %s: %v", vm.ID, err)
			http.Error(w, "VM domain not found", http.StatusNotFound)
			return
		}
		defer domain.Free()

		// Create a stream for the console
		stream, err := conn.NewStream(0)
		if err != nil {
			log.Printf("Serial proxy: failed to create stream: %v", err)
			http.Error(w, "Failed to create console stream", http.StatusInternalServerError)
			return
		}
		defer stream.Finish()

		// Open the serial console (devname is empty for default console)
		// Using DOMAIN_CONSOLE_FORCE to force opening even if in use
		err = domain.OpenConsole("", stream, libvirt.DOMAIN_CONSOLE_FORCE)
		if err != nil {
			log.Printf("Serial proxy: failed to open console for %s: %v", vm.ID, err)
			http.Error(w, fmt.Sprintf("Failed to open serial console: %v", err), http.StatusBadGateway)
			return
		}

		// Upgrade to WebSocket
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{"binary"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("Serial proxy: WebSocket upgrade failed: %v", err)
			return
		}

		netConn := websocket.NetConn(r.Context(), wsConn, websocket.MessageBinary)

		// Bidirectional copy between WebSocket and stream
		done := make(chan struct{}, 2)

		// Stream -> WebSocket
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stream.Recv(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Serial proxy: stream recv error: %v", err)
					}
					done <- struct{}{}
					return
				}
				if n > 0 {
					if _, err := netConn.Write(buf[:n]); err != nil {
						log.Printf("Serial proxy: WebSocket write error: %v", err)
						done <- struct{}{}
						return
					}
				}
			}
		}()

		// WebSocket -> Stream
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := netConn.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Serial proxy: WebSocket read error: %v", err)
					}
					done <- struct{}{}
					return
				}
				if n > 0 {
					if _, err := stream.Send(buf[:n]); err != nil {
						log.Printf("Serial proxy: stream send error: %v", err)
						done <- struct{}{}
						return
					}
				}
			}
		}()

		// Wait for either direction to finish
		<-done
		stream.Finish()
	}
}
