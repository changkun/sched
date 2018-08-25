package pq

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	id        string
	execution time.Time
}

func (t *Task) GetID() (id string) {
	id = t.id
	return
}
func (t *Task) GetExecution() (execute time.Time) {
	execute = t.execution
	return
}
func (t *Task) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}
func (t *Task) GetRetryDuration() (duration time.Duration) {
	return time.Second
}
func (t *Task) SetID(id string) {
	t.id = id
}
func (t *Task) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}
func (t *Task) Execute() (retry bool, fail error) {
	fmt.Println("queue task execute: ", t.id)
	return false, nil
}
func TestTaskQueue(t *testing.T) {
	// Some items and their priorities.
	start := time.Now().UTC()
	pqueue := NewTimerTaskQueue()

	// Insert a new item and then modify its priority.
	task := &Task{
		id:        uuid.Must(uuid.NewRandom()).String(),
		execution: start.Add(time.Millisecond * 5),
	}
	pqueue.Push(task)

	// Insert a new item and then modify its priority.
	for i := 0; i < 30000; i++ {
		fmt.Println("push peek pop ", i)
		task = &Task{
			id:        uuid.Must(uuid.NewRandom()).String(),
			execution: start.Add(time.Millisecond * 5),
		}
		go pqueue.Push(task)
		go pqueue.Peek()
		go pqueue.Update(task)
		go pqueue.Pop()
	}

	fmt.Printf("peek: %v\n", pqueue.Peek())
	// Take the items out; they arrive in decreasing priority order.
	count := 0
	for pqueue.Len() > 0 {
		task := pqueue.Pop()
		count++
		fmt.Printf("pop: %v, count: %d\n", task, count)
	}
}
