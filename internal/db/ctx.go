package dbutil

import (
	"context"
	"time"
)

const defaultTimeout = 3 * time.Second

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}
