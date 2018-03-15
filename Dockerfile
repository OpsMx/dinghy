FROM alpine:3.6

EXPOSE 8081

WORKDIR /usr/local/bin/

ADD ./build/main /usr/local/bin/main
RUN apk update
RUN apk add ca-certificates

ENTRYPOINT ["/usr/local/bin/main"]