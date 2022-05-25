rtmp:
	podman run --rm --network=host tiangolo/nginx-rtmp

bin:
	mkdir bin

zestt: $(wildcard *.go) bin
	CGO_ENABLED=0 go build -o bin/zestt -v .

run: zestt
	./bin/zestt
