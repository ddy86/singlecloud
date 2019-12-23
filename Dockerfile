FROM golang:1.12.5-alpine3.9 AS build

ARG version
ARG buildtime

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud
WORKDIR /go/src/github.com/zdnscloud/singlecloud
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime" cmd/singlecloud/singlecloud.go

FROM scratch
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /
ENTRYPOINT ["/singlecloud"]
