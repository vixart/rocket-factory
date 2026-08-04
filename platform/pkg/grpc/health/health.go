package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Server implements the gRPC Health Checking Protocol (https://github.com/grpc/grpc/blob/master/doc/health-checking.md).
//
// It is the standard way to probe the state of a gRPC service, used by:
//   - Kubernetes: livenessProbe and readinessProbe, via grpc-health-probe or the native gRPC probe
//     (spec.containers[].livenessProbe.grpc.port). Kubelet calls Check periodically: if the service
//     does not answer SERVING, the Pod is restarted (liveness) or removed from balancing (readiness)
//   - gRPC load balancers (Envoy, grpc-go client-side balancing) to find the available backends
//   - monitoring systems and health-check dashboards
//
// The current implementation always returns SERVING: the service counts as healthy as long as
// the gRPC server accepts connections. For richer scenarios, dependencies can be added to the
// struct (database pool, Redis client, ...) and probed inside Check:
//
//	type Server struct {
//	    grpc_health_v1.UnimplementedHealthServer
//	    db    *pgxpool.Pool   // pool.Ping(ctx) to probe PostgreSQL
//	    redis *redis.Client   // client.Ping(ctx) to probe Redis
//	}
//
// If any dependency does not respond, return NOT_SERVING and Kubernetes will restart the
// Pod (liveness) or take it out of balancing (readiness).
type Server struct {
	grpc_health_v1.UnimplementedHealthServer
}

// Check is the unary RPC that reports service health.
//
// It is called by Kubernetes (gRPC liveness/readiness probes), load balancers and clients,
// and returns one of SERVING, NOT_SERVING, UNKNOWN.
// req.Service allows probing a specific service; an empty string means the whole server.
func (s *Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch is the server-streaming RPC that pushes health status changes.
//
// Unlike Check, which is pull based, Watch lets a client receive updates in real time
// without polling. gRPC load balancers use it to react to backend state changes
// immediately.
// The current implementation sends SERVING once and closes the stream.
func (s *Server) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}

// RegisterService registers the health service on a gRPC server.
// Once registered it is reachable at the standard grpc.health.v1.Health path.
func RegisterService(s *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(s, &Server{})
}
