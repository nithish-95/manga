# --- Stage 1: Build the Frontend CSS ---
# Use a Node.js image to build the frontend assets
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy only the necessary package files first to leverage Docker caching
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install

# Copy the rest of the frontend source code
COPY frontend/ ./

# Build the CSS file
RUN npm run build:css


# --- Stage 2: Build the Go Backend ---
# Use a Go image to build the backend binary
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app

# Copy Go module files and download dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the backend source code
COPY backend/ ./

# Build the Go application, placing the binary in the root
RUN go build -o /manga ./main.go


# --- Stage 3: Create the Final Production Image ---
# Use a minimal base image for the final product
FROM alpine:latest
WORKDIR /app

# IMPORTANT: Create the directories before copying files into them
RUN mkdir -p /app/bin && \
    mkdir -p /app/frontend/public/css && \
    mkdir -p /app/frontend/templates

# Copy the compiled Go binary from the backend-builder stage
COPY --from=backend-builder /manga /app/bin/manga

# Copy the built CSS file from the frontend-builder stage
COPY --from=frontend-builder /app/frontend/public/css/output.css /app/frontend/public/css/output.css

# Copy the HTML templates
COPY frontend/templates ./frontend/templates

# Expose the port your Go application listens on
EXPOSE 3001

# The command to run your application
CMD ["/app/bin/manga"]