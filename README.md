# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![codecov](https://codecov.io/gh/changkun/goscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/goscheduler) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)
[![](https://img.shields.io/badge/language-English-blue.svg)](./README.md) [![](https://img.shields.io/badge/language-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-red.svg)](./README_cn.md) 

`goscheduler` is a consistently reliable embedded task scheduler library for _GO_, which applies to be a microkernel of an internal application service, and pluggable tasks must implements `goscheduler` Task interface.

`goscheduler ` not only schedules a task at a specific time or reschedules a planned task immediately, but also flexible to support periodically tasks, which differ from traditional non-consistently unreliable cron task scheduling.

Furthermore, `goscheduler` uses priority queue schedules all tasks and a distributed lock mechanism that ensures tasks can only be executed once across multiple replica instances.

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

## Benchmarks

See [benchmarks](./benchmarks/bench.md) getting to know more analysis of `goscheduler` performance.

## License

[MIT](./LICENSE) &copy; [Changkun Ou](https://changkun.de)