package health

import "context"

// HealthPinger can be implemented by components to expose a specialized
// health check. HealthPing must return nil when the component is healthy.
type HealthPinger interface {
	HealthPing(ctx context.Context) error
}
