all: *.go runtime/*.go runtime/*.s
	go build

test:
	go build -o babygo
	./babygo < testcases/hello.go > tmp/hello.s
	as -o tmp/hello.o tmp/hello.s runtime/runtime.s
	ld -o tmp/hello tmp/hello.o
	./tmp/hello

self:
# 1st generation
	go build -o babygo
# 2nd generation
	./babygo < *.go > tmp/babygo2.s
	as -o tmp/babygo2.o tmp/babygo2.s runtime/runtime.s
	ld -o ./babygo2 tmp/babygo2.o
# 3nd generation
	./babygo2 < *.go > tmp/babygo3.s
	as -o tmp/babygo3.o tmp/babygo3.s runtime/runtime.s
	diff -s tmp/babygo2.s tmp/babygo3.s

clean:
	rm ./babygo
	rm -f tmp/*

.PHONY: test self-test clean
