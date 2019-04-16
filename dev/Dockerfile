FROM alpine:3.9
RUN apk --update add ca-certificates curl

COPY provision /bin/

WORKDIR /

ENTRYPOINT ["/bin/provision"]