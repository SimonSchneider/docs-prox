FROM golang:1.14.0-stretch AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=1

WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
# RUN env GOOS=linux GOARCH=amd64 go build -a -o docs-prox ./cmd/*.go
RUN go build -o docs-prox ./cmd/*.go

# Copy or create other directories/files your app needs during runtime.
# E.g. this example uses /data as a working directory that would probably
#      be bound to a perstistent dir when running the container normally
RUN mkdir /data

# Create the minimal runtime image
FROM scratch

COPY --chown=0:0 --from=builder /docs-prox /

# Set up the app to run as a non-root user inside the /data folder
# User ID 65534 is usually user 'nobody'. 
# The executor of this image should still specify a user during setup.
COPY --chown=65534:0 --from=builder /data /data
USER 65534
WORKDIR /data

ENTRYPOINT ["/test"]