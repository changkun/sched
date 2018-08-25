# `goscheduler` Benchmarks

[![](https://img.shields.io/badge/language-English-blue.svg)](./bench.md) [![](https://img.shields.io/badge/language-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-red.svg)](./bench_cn.md) 

There is a benchmark senario in [./main.go](./main.go), in detail:

1. Every scheduled task will be executed `retry` times
2. There are number of `total` tasks with randomly execution time
3. Total execution time doesn't exceed 10 seconds

## Conclusions

1. Precision of scheduling depends on Go `time.Timer` and interacting performance with Redis
2. In a extensive task scheduling senario, `goscheduler` meets a small delay, but still reliable
3. `goscheduler` minimizes running goroutines during scheduling

## 2 Tasks

```
task task1 is scheduled: hello world!, retry: 0, execution tolerance: 198.266636ms
task task0 is scheduled: hello world!, retry: 0, execution tolerance: 4.49302ms
task task0 is scheduled: hello world!, retry: 1, execution tolerance: 2.229775ms
task task1 is scheduled: hello world!, retry: 1, execution tolerance: 196.243188ms
task task0 is scheduled: hello world!, retry: 2, execution tolerance: 1.99117ms
task task1 is scheduled: hello world!, retry: 2, execution tolerance: 196.036684ms
```

Final report:

```
                   6 Execution in 194ms
-------------------------------------------------------
        required execution:  6
          actual execution:  6
-------------------------------------------------------
   first required schedule:  Aug 25 12:34:44.230631001
     first actual schedule:  Aug 25 12:34:44.430497056
-------------------------------------------------------
    last required schedule:  Aug 25 12:34:44.424631001
      last actual schedule:  Aug 25 12:34:46.426880348
-------------------------------------------------------
      first schedule delay:  199.866055ms
       last schedule delay:  2.002249347s
-------------------------------------------------------
required execution density:  32.333333ms
  actual execution density:  332.730548ms
-------------------------------------------------------
```

![](./images/2.png)

## 100 Tasks

```
task task85 is scheduled: hello world!, retry: 0, execution tolerance: 3.076097021s
task task63 is scheduled: hello world!, retry: 0, execution tolerance: 3.124268845s
task task31 is scheduled: hello world!, retry: 0, execution tolerance: 2.872270863s
task task8 is scheduled: hello world!, retry: 0, execution tolerance: 2.824283833s
task task20 is scheduled: hello world!, retry: 0, execution tolerance: 2.785680855s
task task43 is scheduled: hello world!, retry: 0, execution tolerance: 2.717754923s
task task96 is scheduled: hello world!, retry: 0, execution tolerance: 2.728890534s
task task81 is scheduled: hello world!, retry: 0, execution tolerance: 2.329547213s
task task37 is scheduled: hello world!, retry: 0, execution tolerance: 2.6518907s
task task10 is scheduled: hello world!, retry: 0, execution tolerance: 2.589122199s
task task85 is scheduled: hello world!, retry: 1, execution tolerance: 2.224652378s
task task63 is scheduled: hello world!, retry: 1, execution tolerance: 2.127684472s
task task82 is scheduled: hello world!, retry: 0, execution tolerance: 2.146698817s
task task16 is scheduled: hello world!, retry: 0, execution tolerance: 2.073186977s
task task5 is scheduled: hello world!, retry: 0, execution tolerance: 1.967132954s
task task49 is scheduled: hello world!, retry: 0, execution tolerance: 1.932168323s
...
```

Final report:

```
                   30 Execution in 7.625s
-------------------------------------------------------
        required execution:  30
          actual execution:  30
-------------------------------------------------------
   first required schedule:  Aug 25 11:54:33.403801310
     first actual schedule:  Aug 25 11:54:36.253730350
-------------------------------------------------------
    last required schedule:  Aug 25 11:54:41.028801310
      last actual schedule:  Aug 25 11:54:43.032104997
-------------------------------------------------------
      first schedule delay:  2.84992904s
       last schedule delay:  2.003303687s
-------------------------------------------------------
required execution density:  254.166666ms
  actual execution density:  225.945821ms
-------------------------------------------------------
```

![](./images/100.png)

## 1000 Tasks

```
task task464 is scheduled: hello world!, retry: 0, execution tolerance: 1.652250694s
task task447 is scheduled: hello world!, retry: 0, execution tolerance: 2.023717728s
task task85 is scheduled: hello world!, retry: 0, execution tolerance: 2.232905768s
task task235 is scheduled: hello world!, retry: 0, execution tolerance: 2.232690631s
task task477 is scheduled: hello world!, retry: 0, execution tolerance: 2.218390923s
task task157 is scheduled: hello world!, retry: 0, execution tolerance: 2.204110785s
task task546 is scheduled: hello world!, retry: 0, execution tolerance: 2.190690293s
task task308 is scheduled: hello world!, retry: 0, execution tolerance: 2.189265002s
...
task task834 is scheduled: hello world!, retry: 0, execution tolerance: 985.6791ms
task task43 is scheduled: hello world!, retry: 1, execution tolerance: 989.662571ms
task task70 is scheduled: hello world!, retry: 0, execution tolerance: 994.910615ms
task task789 is scheduled: hello world!, retry: 1, execution tolerance: 999.626984ms
task task266 is scheduled: hello world!, retry: 1, execution tolerance: 989.777293ms
...
```

Final report:

```
                   3000 Execution in 9.985s
-------------------------------------------------------
        required execution:  3000
          actual execution:  3000
-------------------------------------------------------
   first required schedule:  Aug 25 11:58:49.704405062
     first actual schedule:  Aug 25 11:58:51.593286245
-------------------------------------------------------
    last required schedule:  Aug 25 11:58:59.689405062
      last actual schedule:  Aug 25 11:59:01.690042098
-------------------------------------------------------
      first schedule delay:  1.888881183s
       last schedule delay:  2.000637036s
-------------------------------------------------------
required execution density:  3.328333ms
  actual execution density:  3.365585ms
-------------------------------------------------------
```

![](./images/1000.png)