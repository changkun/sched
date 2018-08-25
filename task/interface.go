package task

import "time"

// Interface for a schedulable
type Interface interface {
	GetID() (id string)
	GetExecution() (execute time.Time)
	GetTimeout() (executeTimeout time.Duration)
	GetRetryDuration() (duration time.Duration)
	SetID(id string)
	SetExecution(new time.Time) (old time.Time)
	Execute() (retry bool, fail error)
}
