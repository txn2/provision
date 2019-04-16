FROM alpine:3.9 as certs
RUN apk --update add ca-certificates

FROM scratch
ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY provision /bin/

WORKDIR /

ENTRYPOINT ["/bin/provision"]