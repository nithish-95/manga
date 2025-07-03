# --- Stage 1: Build the Frontend CSS ---
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build:css


# --- Stage 2: Build the self-contained Go Backend ---
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app

# Copy Go module files and download dependencies
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy source code for backend and frontend
COPY backend/ ./backend
COPY frontend/ ./frontend

# IMPORTANT: Replace the source CSS with the one we just built
COPY --from=frontend-builder /app/frontend/public/css/output.css ./frontend/public/css/output.css

# Build the Go application. The 'embed' directives will bundle the frontend files.
RUN cd backend && go build -o /manga .


# --- Stage 3: Create the Final Production Image ---
FROM alpine:latest
WORKDIR /app

# Copy ONLY the single, self-contained binary from the builder stage
COPY --from=backend-builder /manga .

# Expose the port your Go application listens on
EXPOSE 3001

# The command to run your application
CMD ["./manga"]