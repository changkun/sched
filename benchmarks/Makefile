bench: clean
	go build -o bench sched.go
	./bench

trace:
	go tool trace bench.trace

clean:
	rm -rf bench bench.trace