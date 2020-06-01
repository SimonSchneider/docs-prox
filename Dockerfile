FROM golang:1.14.0-stretch AS builder

# ENV GO111MODULE=on \
#   CGO_ENABLED=0 \
#   GOARCH=amd64 \
#   GOOS=linux

WORKDIR /build
# COPY go.mod .
# COPY go.sum .
# RUN go mod download
COPY . .
# RUN go build -a -o docs-prox ./cmd/*.go

RUN mkdir /data
RUN cp -r _config /data/

FROM scratch
COPY --chown=0:0 --from=builder /build/docs-prox /
COPY --chown=65534:0 --from=builder /data /data
USER 65534
WORKDIR /data
ENTRYPOINT ["/docs-prox"]