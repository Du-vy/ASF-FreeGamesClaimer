# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod ./
COPY main.go ./

# Compile statically-linked binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o asfclaim main.go

# Stage 2: Final lightweight runner
FROM alpine:latest

# Install ca-certificates so HTTPS requests to GitHub API work
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary from builder stage
COPY --from=builder /usr/src/app/asfclaim .

CMD ["./asfclaim"]
