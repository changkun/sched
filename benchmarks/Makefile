bench: clean
	go build -o bench main.go
	./bench

trace:
	go tool trace bench.trace

clean:
	rm -rf bench bench.trace