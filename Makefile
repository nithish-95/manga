all: clean build run

build:
	go build -o bin/manga .
run: 
	bin/manga
clean:
	cd backend && go mod tidy
	rm backend/bin/* || true