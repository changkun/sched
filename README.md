# sched

[![GoDoc](https://godoc.org/github.com/changkun/sched?status.svg)](https://godoc.org/github.com/changkun/sched) [![Build Status](https://travis-ci.org/changkun/sched.svg?branch=master)](https://travis-ci.org/changkun/sched) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/sched)](https://goreportcard.com/report/github.com/changkun/sched) [![codecov](https://codecov.io/gh/changkun/sched/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/sched) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fchangkun%2Fsched.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fchangkun%2Fsched?ref=badge_shield)
[![](https://img.shields.io/github/release/changkun/sched/all.svg)](https://github.com/changkun/sched/releases)
[![](https://img.shields.io/badge/language-English-blue.svg)](./README.md) [![](https://img.shields.io/badge/language-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-red.svg)](./README_cn.md) 

`sched` is a consistently reliable embedded task scheduler library for _GO_. It applies to be a microkernel of an internal application service, and pluggable tasks must implements `sched` **Task** interface.

`sched ` not only schedules a task at a specific time or reschedules a planned task immediately, but also flexible to support periodically tasks, which differ from traditional non-consistently unreliable cron task scheduling.

Furthermore, `sched` manage tasks, like goroutine runtime scheduler, uses priority queue schedules all tasks and a distributed lock mechanism that ensures tasks can only be executed once across multiple replica instances.

## Features

- **Flexible Scheduling** 
  - Single execution, period-equally execution, period-inequally execution
- **Microkernel Embedding**
  - Embedding into an application without change existing code
- **Distributed Reliability**
  - A task can only be executed once across replica instances
- **Eventually Consistency**
  - All tasks that scheduled must be executed eventually
- **Fault Tolerance**
  - Recover when restart, retry if needed or on error

## Getting started

```go
// Init sched database
sched.Init("redis://127.0.0.1:6379/1")

// Recover tasks
sched.Recover(&ArbitraryTask1{}, &ArbitraryTask2{})

// Setup tasks
sched.Setup(&ArbitraryTask1{...}, &ArbitraryTask2{...})

// Launch a task
sched.Launch(&ArbitraryTask1{...}, &ArbitraryTask2{...})
```

## Benchmarks

See [benchmarks](./benchmarks/bench.md) getting to know more analysis of `sched` performance.

## License

[MIT](./LICENSE) &copy; [Changkun Ou](https://changkun.de)

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fchangkun%2Fsched.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fchangkun%2Fsched?ref=badge_large)