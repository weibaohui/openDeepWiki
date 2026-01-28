FROM golang:alpine AS builder
WORKDIR /app
COPY . /app
RUN ls -al /app/backend
RUN cd /app/backend && go mod download && CGO_ENABLED=1  go build -ldflags "-s -w " -o /app/server ./cmd/server/

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server /app/server
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk upgrade && apk add --no-cache ca-certificates  \
    && apk del alpine-conf && rm -rf /var/cache/* && chmod +x server

#Server
EXPOSE 8080
CMD [ "/app/server"]
