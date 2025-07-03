all: clean build run

build:
	cd backend && go build -o ../bin/manga .
	cd frontend && \
	npm install && \
	echo "--- START DEBUGGING ---" && \
	ls -la ./node_modules/.bin/ && \
	echo "--- FILE CONTENT ---" && \
	cat ./node_modules/.bin/tailwindcss && \
	echo "--- END DEBUGGING ---" && \
	npm run build:css

run: 
	bin/manga

clean:
	cd backend && go mod tidy
	rm backend/bin/* || true