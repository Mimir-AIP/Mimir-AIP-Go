package execution

import "context"

// Backend runs queued asynchronous work using a specific execution environment.
type Backend interface {
	Run(ctx context.Context)
}
