# Start from golang base image
FROM golang:alpine as builder

# ENV GO111MODULE=on

# Add Maintainer info
LABEL maintainer="Alex Volkov"

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git \
&& rm -rf /var/lib/apt/lists/*

# Set the current working directory inside the container 
WORKDIR /ocm-description-svc

# Copy go mod and sum files 
COPY go.mod go.sum ./

# RUN go mod tidy

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed 
RUN go mod download 

# Copy the source from the current directory to the working Directory inside the container 

COPY . .
# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Start a new stage from scratch
FROM alpine:3.17
RUN apk --no-cache add ca-certificates

RUN addgroup -S etsn && adduser -S etsn -G etsn -u 1001
USER etsn


WORKDIR /home/etsn/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /ocm-description-svc/main .
# COPY --from=builder /ocm-description-service/.env .

# Expose port 8080 to the outside world
EXPOSE 8083

#Command to run the executable
CMD ["./main"]