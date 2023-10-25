ARG IMAGE=alpine:latest

FROM golang:alpine AS build
WORKDIR /
RUN apk add gcc musl-dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest

FROM $IMAGE
COPY --from=build /go/bin/dlv /bin/dlv
ENTRYPOINT ["/bin/sh", "-c", "while true; do printf '%s %s\n' \"$(date -u)\" \"handling gograpple debug session\"; sleep 36000; done"]