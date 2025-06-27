FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-ceph-backup .

FROM alpine:latest

# Install required packages
RUN apk --no-cache add ca-certificates ceph-common gnupg

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/k8s-ceph-backup .

# Copy entrypoint script
COPY entrypoint.sh .

# Create directories for temporary files and configuration
RUN mkdir -p /tmp/k8s-ceph-backup /etc/ceph
RUN mkdir -p /tmp/gnupg && chmod 600 /tmp/gnupg
# Set permissions
RUN chmod +x k8s-ceph-backup entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
