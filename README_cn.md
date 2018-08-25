# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![codecov](https://codecov.io/gh/changkun/goscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/goscheduler) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)
[![](https://img.shields.io/badge/language-English-blue.svg)](./README.md) [![](https://img.shields.io/badge/language-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-red.svg)](./README_cn.md) 

`goscheduler` 是一个 _GO_ 编写的一致可靠的嵌入型任务调度库，适合作为应用服务内部核心任务调度的一个微内核，任务插件通过实现 `goscheduler` 所定义的接口来完成。

摒弃了 `cron` 的周期性不可靠、无容错式调度，`goscheduler` 无需了解 `cron` 调度语法，却比 cron 更加灵活，
不仅能支持单次任务执行调度或重新调度现有任务，亦能支持周期式、不定周期式反复调度。

## 特性

- **可变式调度** 
  - 单次执行、等周期式执行、不等周期式执行
- **微内核设计**
  - 在不侵入原有代码的情况通过实现接口来接入调度
- **分布式可靠**
  - 多个分布式副本节点间只被调度一次
- **最终一致性**
  - 所有任务终将被调度执行
- **强容错机制**
  - 当应用重启、或任务有需要时能被重新调度

## 性能

查看[性能测试](./benchmarks/bench_cn.md)来了解关于 `goscheduler` 的性能分析。

## 许可

[MIT](./LICENSE) &copy; [欧长坤](https://changkun.de)