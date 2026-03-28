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
func GetAuditClient(addr string) (pb.AuditLogServiceClient, error) {
	once.Do(func() {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			initErr = fmt.Errorf("auditclient: failed to connect to %s: %w", addr, err)
			return
		}
		auditClient = pb.NewAuditLogServiceClient(conn)
	})
	return auditClient, initErr
}
