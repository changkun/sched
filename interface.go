package goscheduler

import (
	"time"
)

// Task interface defines the task need to be executed
type Task interface {
	// Identifier should returns a unique string for the task, usually can return an UUID
	Identifier() string
	// GetExecuteTime should returns the excute time of the task
	GetExecuteTime() time.Time
	// SetExecuteTime can set the execution time of the task
	SetExecuteTime(t time.Time) time.Time
	// Execute defines the actual running task
	Execute() error
	// FailRetryDuration returns the task retry duration if fails
	FailRetryDuration() time.Duration
}

// Config provides the database URI for goschduler
type Config struct {
	DatabaseURI string
}

// Init creates the connection of database
func Init(config *Config) error {
	return newScheduler().initDB(config.DatabaseURI)
}

// Poll recover tasks from database when application boot up
func Poll(t Task) error {
	return newScheduler().recover(t)
}

// Schedule a task
func Schedule(t Task) error {
	s := newScheduler()
	if err := s.save(t); err != nil {
		return err
	}

	runner := s.getRunner(t)
	if runner != nil {
		// timer Stop() can fail in the following case:
		//  - a timer has executed: this is not possible in our implementation
		//    because timer will be nil after arrival
		//  - user created a timer directly, without invoking startTimer:
		//    not our case
		// see https://github.com/golang/go/blob/b5375d70d1d626595ec68568b68cc28dddc859d1/src/runtime/time.go#L166
		runner.Stop()
	}
	reschedule(t)
	return nil
}

// Boot a task immediately
func Boot(t Task) error {
	t.SetExecuteTime(time.Now().UTC())
	return Schedule(t)
}
