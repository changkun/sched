# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![codecov](https://codecov.io/gh/changkun/goscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/goscheduler) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)

goschduler is a task scheduler library with data persistence for go.

## Features

- Schedule a task at a specific time
- Boot (an existing) task immediately
- Recover specified type of tasks from database when app restarts
- Auto retry if scheduled task faild

## Usage

goscheduler persists your task to [Redis](https://redis.io/), 
it schedules your task at a specific time or boot an existing task immediately.

See [example](./example/main.go) for detail usage.

## License

[MIT](./LICENSE)