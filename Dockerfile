FROM golang:1.14.0-stretch AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=1

WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build ./cmd/test.go
WORKDIR /dist
RUN cp /build/test ./test

# Optional: in case your application uses dynamic linking (often the case with CGO), 
# this will collect dependent libraries so they're later copied to the final image
# NOTE: make sure you honor the license terms of the libraries you copy and distribute
RUN ldd test | tr -s '[:blank:]' '\n' | grep '^/' | \
  xargs -I % sh -c 'mkdir -p $(dirname ./%); cp % ./%;'
RUN mkdir -p lib64 && cp /lib64/ld-linux-x86-64.so.2 lib64/

# Copy or create other directories/files your app needs during runtime.
# E.g. this example uses /data as a working directory that would probably
#      be bound to a perstistent dir when running the container normally
RUN mkdir /data

# Create the minimal runtime image
FROM scratch

COPY --chown=0:0 --from=builder /dist /

# Set up the app to run as a non-root user inside the /data folder
# User ID 65534 is usually user 'nobody'. 
# The executor of this image should still specify a user during setup.
COPY --chown=65534:0 --from=builder /data /data
USER 65534
WORKDIR /data

ENTRYPOINT ["/test"]