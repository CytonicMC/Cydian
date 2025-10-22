# Multi-stage Dockerfile for Cydian
# Builder stage
FROM --platform=$BUILDPLATFORM golang:1.25 AS build

WORKDIR /workspace

# Pre-cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build for the target platform
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -a -o Cydian ./main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /workspace/Cydian /

# Run the binary
CMD ["/Cydian"]
