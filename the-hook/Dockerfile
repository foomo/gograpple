ARG IMAGE=busybox
FROM golang:latest AS build
WORKDIR /
# RUN CGO_ENABLED=0 go get -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv
RUN go install github.com/go-delve/delve/cmd/dlv@latest

FROM $IMAGE
COPY --from=build /go/bin/dlv /bin/dlv
ENTRYPOINT ["/bin/sh", "-c", "while true; do echo go-grapple-go!; sleep 3600; done"]