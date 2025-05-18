FROM golang:alpine AS builder
ARG VERSION
ARG GIT_COMMIT
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download
COPY . /app
RUN CGO_ENABLED=0  go build -ldflags "-s -w  -X main.Version=$VERSION -X main.GitCommit=$GIT_COMMIT  -X main.GitTag=$GIT_TAG  -X main.GitRepo=$GIT_REPOSITORY  -X main.BuildDate=$BUILD_DATE  -o /app/openDeepWiki

FROM alpine:latest

WORKDIR /app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache curl bash inotify-tools

COPY --from=builder /app/openDeepWiki /app/openDeepWiki
#openDeepWiki Server
EXPOSE 3721
ENTRYPOINT ["/app/openDeepWiki"]