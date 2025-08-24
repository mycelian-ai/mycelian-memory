package job

import (
	"context"
	"errors"
	"fmt"
)

// ErrNilJobFunc is returned when a JobFunc is nil.
var ErrNilJobFunc = errors.New("nil JobFunc")

// jobFunc lets us pass plain closures to the shard executor.
type jobFunc func(context.Context) error

func (f jobFunc) Run(ctx context.Context) error {
	if f == nil {
		return fmt.Errorf("jobfunc: %w", ErrNilJobFunc)
	}
	return f(ctx)
}

// New creates a new job function from a closure.
func New(fn func(context.Context) error) jobFunc {
	return jobFunc(fn)
}
