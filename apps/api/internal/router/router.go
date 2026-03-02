package router

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/connectsvc"
	"github.com/doomedramen/lab/apps/api/internal/handler"
	appmiddleware "github.com/doomedramen/lab/apps/api/internal/middleware"
	"github.com/doomedramen/lab/apps/api/internal/service"
	"github.com/doomedramen/lab/apps/api/pkg/tus"
)

// Router creates and configures the main router.
// Metrics and events routes (REST) are registered separately in main.go
// because they depend on optional SQLite repositories.
func Router(
	clusterSvc *service.ClusterService,
	nodeSvc *service.NodeService,
	vmSvc *service.VMService,
	containerSvc *service.ContainerService,
	stackSvc *service.StackService,
	stacksDir string,
	isoSvc *service.ISOService,
	tusHandler *tus.Handler,
	authService *service.AuthService,
	snapshotSvc *service.SnapshotService,
	backupSvc *service.BackupService,
	taskSvc *service.TaskService,
	storageSvc *service.StorageService,
	networkSvc *service.NetworkService,
	firewallSvc *service.FirewallService,
	alertSvc *service.AlertService,
	proxySvc *service.ProxyService,
	authInterceptor *appmiddleware.AuthInterceptor,
	healthHandler *handler.HealthHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(appmiddleware.Logging)
	r.Use(appmiddleware.CORS)
	r.Use(middleware.Recoverer)

	// Health check (plain HTTP — not migrated to Connect)
	r.Get("/health", handler.HealthCheck)
	if healthHandler != nil {
		r.Get("/health/ready", healthHandler.ServeHTTP)
	}

	// VNC WebSocket proxy (plain HTTP — not Connect RPC)
	r.Handle("/ws/vnc", handler.VNCProxyHandler(vmSvc))

	// Serial console WebSocket proxy (plain HTTP — not Connect RPC)
	r.Handle("/ws/serial", handler.SerialProxyHandler(vmSvc))

	// Docker container bash PTY (plain HTTP — not Connect RPC)
	r.Handle("/ws/stack-bash", handler.ContainerBashHandler(stackSvc))

	// Docker stack log streaming (plain HTTP — not Connect RPC)
	r.Handle("/ws/stack-logs", handler.StackLogsHandler(stackSvc, stacksDir))

	// Host shell access (plain HTTP — not Connect RPC)
	r.Handle("/ws/host-shell", handler.HostShellHandler(nodeSvc))

	// Tus upload routes (protocol-incompatible with Connect — stays as plain HTTP)
	if tusHandler != nil {
		r.Handle("/tus/*", tusHandler.Middleware(tusHandler))
		r.HandleFunc("/tus", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/tus/", http.StatusMovedPermanently)
		})
	}

	// Connect RPC handlers with optional auth interceptor
	connectMux := http.NewServeMux()

	// Rate limiter for auth endpoints: 5 attempts per second, burst of 10.
	// This prevents brute-force on Login, Register, and MFA endpoints.
	authRateLimiter := appmiddleware.NewRateLimiter(rate.Limit(5), 10)

	// Auth service (always available)
	if authService != nil {
		authHandler := handler.NewAuthServiceServer(authService)
		connectMux.Handle(labv1connect.NewAuthServiceHandler(authHandler,
			connect.WithInterceptors(authRateLimiter.Interceptor(), authInterceptor),
		))
	}

	// Other services with auth interceptor
	interceptors := []connect.Interceptor{}
	if authInterceptor != nil {
		interceptors = append(interceptors, authInterceptor)
	}

	connectMux.Handle(labv1connect.NewClusterServiceHandler(connectsvc.NewClusterServiceServer(clusterSvc), connect.WithInterceptors(interceptors...)))
	connectMux.Handle(labv1connect.NewNodeServiceHandler(connectsvc.NewNodeServiceServer(nodeSvc), connect.WithInterceptors(interceptors...)))
	connectMux.Handle(labv1connect.NewVmServiceHandler(connectsvc.NewVmServiceServer(vmSvc), connect.WithInterceptors(interceptors...)))
	connectMux.Handle(labv1connect.NewContainerServiceHandler(connectsvc.NewContainerServiceServer(containerSvc), connect.WithInterceptors(interceptors...)))
	connectMux.Handle(labv1connect.NewStackServiceHandler(connectsvc.NewStackServiceServer(stackSvc), connect.WithInterceptors(interceptors...)))
	connectMux.Handle(labv1connect.NewIsoServiceHandler(connectsvc.NewIsoServiceServer(isoSvc), connect.WithInterceptors(interceptors...)))
	
	// Snapshot service (if available)
	if snapshotSvc != nil {
		snapshotHandler := handler.NewSnapshotServiceServer(snapshotSvc)
		connectMux.Handle(labv1connect.NewSnapshotServiceHandler(snapshotHandler, connect.WithInterceptors(interceptors...)))
	}

	// Backup service (if available)
	if backupSvc != nil {
		backupHandler := handler.NewBackupServiceServer(backupSvc)
		connectMux.Handle(labv1connect.NewBackupServiceHandler(backupHandler, connect.WithInterceptors(interceptors...)))
	}

	// Task service (if available)
	if taskSvc != nil {
		taskHandler := handler.NewTaskServiceServer(taskSvc)
		connectMux.Handle(labv1connect.NewTaskServiceHandler(taskHandler, connect.WithInterceptors(interceptors...)))
	}

	// Storage service (if available)
	if storageSvc != nil {
		storageHandler := handler.NewStorageServiceServer(storageSvc)
		connectMux.Handle(labv1connect.NewStorageServiceHandler(storageHandler, connect.WithInterceptors(interceptors...)))
	}

	// Network service (if available)
	if networkSvc != nil {
		networkHandler := handler.NewNetworkServiceServer(networkSvc)
		connectMux.Handle(labv1connect.NewNetworkServiceHandler(networkHandler, connect.WithInterceptors(interceptors...)))
	}

	// Firewall service (if available)
	if firewallSvc != nil {
		firewallHandler := handler.NewFirewallServiceServer(firewallSvc)
		connectMux.Handle(labv1connect.NewFirewallServiceHandler(firewallHandler, connect.WithInterceptors(interceptors...)))
	}

	// Alert service (if available)
	if alertSvc != nil {
		alertHandler := connectsvc.NewAlertServiceServer(alertSvc)
		connectMux.Handle(labv1connect.NewAlertServiceHandler(alertHandler, connect.WithInterceptors(interceptors...)))
	}

	// Proxy service (if available)
	if proxySvc != nil {
		proxyHandler := handler.NewProxyServiceServer(proxySvc)
		connectMux.Handle(labv1connect.NewProxyServiceHandler(proxyHandler, connect.WithInterceptors(interceptors...)))
	}

	r.Mount("/", connectMux)

	return r
}
