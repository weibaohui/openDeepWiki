FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download
COPY . /app
RUN CGO_ENABLED=0  go build -ldflags "-s -w  -X main.Version=$VERSION -X main.GitCommit=$GIT_COMMIT  -X main.GitTag=$GIT_TAG  -X main.GitRepo=$GIT_REPOSITORY  " -o /app/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server /app/server
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk upgrade && apk add --no-cache ca-certificates  \
    && apk del alpine-conf && rm -rf /var/cache/* && chmod +x server

#Server
EXPOSE 8080
CMD [ "/app/server"]
