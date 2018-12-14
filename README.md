# sched

[![GoDoc](https://godoc.org/github.com/changkun/sched?status.svg)](https://godoc.org/github.com/changkun/sched) [![Build Status](https://travis-ci.org/changkun/sched.svg?branch=master)](https://travis-ci.org/changkun/sched) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/sched)](https://goreportcard.com/report/github.com/changkun/sched) [![codecov](https://codecov.io/gh/changkun/sched/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/sched) [![](https://img.shields.io/github/release/changkun/sched/all.svg)](https://github.com/changkun/sched/releases)

`sched` is a high performance task scheduling library with future support.

## Getting started

```go
// Init sched, with tasks should recovered when reboot
futures, err := sched.Init("redis://127.0.0.1:6379/1"， &ArbitraryTask1{}, &ArbitraryTask2{})
if err != nil {
    panic(err)
}
// Retrieve task's future
for i := range futures {
    fmt.Printf("%v", futures[i].Get())
}

// Setup tasks, use future.Get() to retrieve the future of task
future, err := sched.Submit(&ArbitraryTask1{...}, &ArbitraryTask2{...})
if err != nil {
    panic(err)
}
fmt.Printf("%v", future.Get())

// Launch a task, use future.Get() to retrieve the future of task
future, err := sched.Trigger(&ArbitraryTask1{...}, &ArbitraryTask2{...})
if err != nil {
    panic(err)
}
fmt.Printf("%v", future.Get())

// Pause scheduling
sched.Pause()

// Resume scheduling
sched.Resume()

// Stop scheduler gracefully
sched.Stop()
```

## License

[MIT](./LICENSE) &copy; [Changkun Ou](https://changkun.de)