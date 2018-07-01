# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![Coverage Status](https://coveralls.io/repos/github/changkun/goscheduler/badge.svg?branch=master)](https://coveralls.io/github/changkun/goscheduler?branch=master) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)

goschduler is a task scheduler library with data persistence for go.

## Features

- Schedule and persist a task at a specific time
- Boot (an existing) task immediately
- Recover specified type of tasks from database when application restart

## Usage

goscheduler persists your task to Redis, it schedules your task at a specific time
or boot an existing task immediately.

See [example](./example/main.go) for detail usage.

## License

[MIT](./LICENSE)