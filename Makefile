all: clean build run

build:
	cd backend && go build -o ../bin/manga .
	cd frontend && npm install && npm run build:css

run: 
	bin/manga

clean:
	cd backend && go mod tidy
	rm backend/bin/* || true
