.PHONY: build-frontend build-backend build run clean

# Build frontend
build-frontend:
	npm install
	npm run build

# Build backend
build-backend:
	go build -o gemgo .

# Build both frontend and backend
build: build-frontend build-backend

# Run the application
run: build
	./gemgo

# Clean build artifacts
clean:
	rm -rf dist/
	rm -f gemgo

# Development mode (if you want to run frontend separately)
dev-frontend:
	npm run dev

dev-backend:
	go run .