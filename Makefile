all: clean build run

build:
	go build -o bin/manga .
run: 
	bin/manga
clean:
	go mod tidy
	rm bin/* || true