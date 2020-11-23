# Start from golang base image
FROM clearlinux/golang:1.15 as builder

# Set the current working directory inside the container
WORKDIR /build

# Copy go.mod, go.sum files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy sources to the working directory
COPY . .

# Build the Go app
# ARG project
RUN GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -a -v -o server ./cmd/sfu

# Start a new stage from busybox
FROM clearlinux:latest

WORKDIR /dist

# Copy the build artifacts from the previous stage
COPY --from=builder /build/server .

# Run the executable
CMD ["./server"]
