# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![codecov](https://codecov.io/gh/changkun/goscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/goscheduler) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)
[![](https://img.shields.io/badge/language-English-blue.svg)](./README.md) [![](https://img.shields.io/badge/language-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-red.svg)](./README_cn.md) 

`goscheduler` 是一个 _GO_ 编写的一致可靠的任务调度库，适合作为应用服务内部核心任务调度的一个微内核，任务插件通过实现 `goscheduler` 所定义的接口来完成。

不同于传统的 cron 周期性不可靠、无容错式调度，`goscheduler` 无需了解 cron 调度语法，却比 cron 更加灵活，
不仅能支持单次任务执行调度或重新调度现有任务，亦能支持周期式反复调度。

此外，`goscheduler` 还使用了分布式锁机制保证了多个副本节点的
分布式任务调度只有一个节点执行所需任务。

## 特性

- 线程安全
- 分布式一致
- 失败重试
- 重启时恢复任务
- 在特定时间点上执行任务
- 立即执行一个（已经存在）的任务

## 用法

goscheduler 使用 [Redis](https://redis.io/) 进行数据持久，在一个特定时间点上调度需要执行的任务

```go
package main

import (
	"fmt"
	"time"

	"github.com/changkun/goscheduler"
)

// CustomTask 定义了你需要执行的任务结构体
type CustomTask struct {
	ID          string    `json:"uuid"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Information string    `json:"info"`
}

// Identifier 必须返回一个全局唯一的字符串，通常返回一个 UUID 即可
func (c CustomTask) Identifier() string {
	return c.ID
}

// GetExecuteTime 必须返回任务的执行时间
func (c CustomTask) GetExecuteTime() time.Time {
	return c.End
}

// SetExecuteTime 提供给 goscheduler 设置任务的执行时间
func (c *CustomTask) SetExecuteTime(t time.Time) time.Time {
	c.End = t
	return c.End
}

// Execute 定义了任务实际运行时的执行方法，当执行失败时，
// 应返回 error 来告知 goscheduler 。
func (c *CustomTask) Execute() error {
	// implement your task execution
	fmt.Println("Task is Running: ", c.Information)
	return nil
}

// FailRetryDuration 返回当任务执行失败时，重试的时间间隔
func (c CustomTask) FailRetryDuration() time.Duration {
	return time.Second
}

func main() {
	// 初始化 goscheduler 数据库
	goscheduler.Init(&goscheduler.Config{
		DatabaseURI: "redis://127.0.0.1:6379/8",
	})

	// 当 goscheduler 数据库初始化完毕后，
	// 可以调用 Poll 来恢复尚未完成调度的任务
	var task CustomTask
	goscheduler.Poll(&task)
	// task 变量仅用于 goscheduler 内部推导需要调度的任务类型
	// Poll 调用完毕后 task 变量仍为零值

	// 创建一个需要执行的任务
	task = CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(10) * time.Second),
		Information: "this is a task message message",
	}
	fmt.Println("Retry duration if execution failed: ", task.FailRetryDuration())

	// 首次将任务调度为十秒后执行
	goscheduler.Schedule(&task)
	// 由于特殊情况，我们决定立即执行此任务
	goscheduler.Boot(&task)

	// 等待调度的任务结束执行
	time.Sleep(time.Second * 2)
}
```

## License

[MIT](./LICENSE)