package needle

import (
	"time"
)

type ResolveHook func(key string, duration time.Duration, err error)

type ProvideHook func(key string)

type StartHook func(key string, duration time.Duration, err error)

type StopHook func(key string, duration time.Duration, err error)
