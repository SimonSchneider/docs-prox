FROM golang:alpine as build
RUN apk --no-cache add ca-certificates
RUN mkdir -p /files

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY swagger_demo.url /config/files/swagger_demo.url
COPY --chown=0:0 main /
VOLUME ["/config"]
COPY config.json /config/config.json
COPY dist dist
USER 65534
EXPOSE 10021
ENV CONFIG_FILE=config/config.json
ENTRYPOINT ["/main"]
