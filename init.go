package auditclient

import (
	"fmt"
	"sync"

	"github.com/sw5005-sus/ceramicraft-audit-client/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	auditClient pb.AuditLogServiceClient
	once        sync.Once
	initErr     error
)

// GetAuditClient returns a singleton AuditLogServiceClient.
// The connection is established on the first call using the provided address.
// Subsequent calls ignore addr and return the already-initialised client.
func GetAuditClient(host string, port int) (pb.AuditLogServiceClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	once.Do(func() {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(1024 * 1024)),
		}
		conn, err := grpc.NewClient(addr, opts...)
		if err != nil {
			initErr = fmt.Errorf("auditclient: failed to connect to %s: %w", addr, err)
			return
		}
		auditClient = pb.NewAuditLogServiceClient(conn)
	})
	return auditClient, initErr
}
