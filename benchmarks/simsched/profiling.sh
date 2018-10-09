FILENAME=$(basename $(pwd))
go test -run=. -bench=. -cpuprofile=cpu.out -benchmem -memprofile=mem.out -trace trace.out
go tool pprof -pdf $FILENAME.test cpu.out > cpu.pdf && open cpu.pdf
go tool pprof -pdf --alloc_space $FILENAME.test mem.out > alloc_space.pdf && open alloc_space.pdf
go tool pprof -pdf --alloc_objects $FILENAME.test mem.out > alloc_objects.pdf && open alloc_objects.pdf
go tool pprof -pdf --inuse_space $FILENAME.test mem.out > inuse_space.pdf && open inuse_space.pdf
go tool pprof -pdf --inuse_objects $FILENAME.test mem.out > inuse_objects.pdf && open inuse_objects.pdf
go tool trace trace.out

# go-torch $FILENAME.test cpu.out -f ${FILENAME}_cpu.svg && open ${FILENAME}_cpu.svg
# go-torch --alloc_objects $FILENAME.test mem.out -f ${FILENAME}_alloc_obj.svg && open ${FILENAME}_alloc_obj.svg
# go-torch --alloc_space $FILENAME.test mem.out -f ${FILENAME}_alloc_space.svg && open ${FILENAME}_alloc_space.svg
# go-torch --inuse_objects $FILENAME.test mem.out -f ${FILENAME}_inuse_obj.svg && open ${FILENAME}_inuse_obj.svg
# go-torch --inuse_space $FILENAME.test mem.out -f ${FILENAME}_inuse_space.svg && open ${FILENAME}_inuse_space.svg