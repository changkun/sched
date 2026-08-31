# sched

[![Go Reference](https://pkg.go.dev/badge/github.com/changkun/sched.svg)](https://pkg.go.dev/github.com/changkun/sched)
[![CI](https://github.com/changkun/sched/actions/workflows/ci.yml/badge.svg)](https://github.com/changkun/sched/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/changkun/sched)](https://goreportcard.com/report/github.com/changkun/sched)
[![codecov](https://codecov.io/gh/changkun/sched/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/sched)
[![Release](https://img.shields.io/github/v/release/changkun/sched)](https://github.com/changkun/sched/releases)

`sched` runs a task at a given time, keeps it across restarts, and runs it
once across every replica of your application.

It is a library, not a service. A task is a Go type that carries its own data
and its own code, so scheduling one is a function call, and the result comes
back through a future.

```
go get github.com/changkun/sched
```

## Quick start

```go
// Restore the tasks that the last run left behind. Pass an empty value of
// every task type you want back, so sched knows what to decode into.
futures, err := sched.Init("redis://127.0.0.1:6379/0", &EmailTask{})
if err != nil {
    return err
}
for _, f := range futures {
    fmt.Println(f.Get())
}

// Schedule a task for the time it reports.
f, err := sched.Submit(&EmailTask{To: "hi@changkun.de", At: tomorrow})
if err != nil {
    return err
}

// Get blocks until the task ran.
fmt.Println(f.Get())

// Or wait without blocking forever.
select {
case <-f.Done():
    fmt.Println(f.Get())
case <-ctx.Done():
    return ctx.Err()
}

sched.Stop() // when the application shuts down
```

Submitting an identifier that is already scheduled moves the task to the new
time instead of adding a second one, and resolves every future waiting for it.
`sched.Trigger` schedules a task for now, `sched.Pause` and `sched.Resume`
stop and start the scheduler, and `sched.Wait` blocks until nothing is queued
or running.

## Writing a task

A task carries its own state, so every field it needs after a restart has to
be exported and encodable by `encoding/json`. sched restores the identifier
and the execution time itself.

```go
type EmailTask struct {
    To string    `json:"to"`
    At time.Time `json:"at"`

    id string
}

func (t *EmailTask) ID() string               { return t.id }
func (t *EmailTask) SetID(id string)          { t.id = id }
func (t *EmailTask) Execution() time.Time     { return t.At }
func (t *EmailTask) SetExecution(e time.Time) { t.At = e }
func (t *EmailTask) Timeout() time.Duration   { return time.Minute }
func (t *EmailTask) RetryTime() time.Time     { return time.Now().UTC().Add(time.Minute) }

func (t *EmailTask) Execute() (any, bool, error) {
    if err := send(t.To); err != nil {
        return nil, true, err // run again at RetryTime
    }
    return "sent", false, nil
}
```

`Timeout` is how long the lock that keeps two replicas apart stays alive.
sched releases it as soon as `Execute` returns, so the lifetime only matters
when the replica dies mid-task. Give it more than `Execute` normally needs: a
lifetime that expires during execution lets a second replica start the same
task, and one much longer than that keeps the task unclaimable after a crash.

A task that panics does not take the process down: the panic comes back
through the future as an `error`. A future always resolves. If sched refuses
to run a task, `Get` returns `ErrTaskClaimed` (another replica took it),
`ErrTaskUnverifiable` (its record could not be read) or `ErrStopped` (the
scheduler shut down first), so no caller waits for a result that cannot
arrive.

## What it does for you

- **Survives restarts.** Every scheduled task is written to Redis before it is
  queued. `Init` brings back what did not run.
- **Runs once across replicas.** Each task takes a Redis lock with a lifetime
  of `Timeout` before it runs.
- **Retries.** A task that returns an error, or asks for a retry, comes back at
  `RetryTime`.
- **Returns a result.** `Submit` and `Trigger` hand back a `Future`.
- **Reacts to other replicas.** A replica rereads the persisted execution time
  before it runs a task, and reschedules if another replica moved it.

## How it works

One goroutine owns a priority queue of tasks ordered by execution time, and a
single timer serves the head of that queue. Callers never touch the queue.
`Submit` first persists the task, which is a round-trip to Redis, and then
hands it to the scheduler through a wait-free multi-producer queue: a fixed
number of atomic operations, with no loop and no retry, so no caller ever
waits for a lock or for another caller. `Pause`, `Resume` and the retry path
use the same handoff. The package holds no mutex.

Each task that comes due runs in its own goroutine, so a slow task delays
neither the scheduler nor the tasks behind it.

```
Submit ─┐
Trigger ─┼─▶ intake (wait-free MPSC) ─▶ scheduler goroutine ─┬─▶ task goroutine ─▶ Future
Pause   ─┘                                 owns queue+timer  └─▶ task goroutine ─▶ Future
```

## Requirements

Go 1.27 and a Redis server. The test suite runs against an in-process Redis
and needs no server of its own.

## License

[MIT](./LICENSE) &copy; [Changkun Ou](https://changkun.de)
