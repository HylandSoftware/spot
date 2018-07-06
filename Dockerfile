FROM golang:1.10 AS builder

ADD . /go/src/github.com/hylandsoftware/spot
WORKDIR /go/src/github.com/hylandsoftware/spot
RUN go get -u github.com/golang/dep/...
RUN make restore && \
    make test && \
    CGO_ENABLED=0 GOOS=linux make build-unix

FROM alpine
COPY --from=builder /go/src/github.com/hylandsoftware/spot/dist/spot /spot

ENTRYPOINT ["/spot"]