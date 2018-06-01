FROM golang AS builder

ADD . /go/src/bitbucket.hylandqa.net/do/spot
WORKDIR /go/src/bitbucket.hylandqa.net/do/spot
RUN go get -u github.com/golang/dep/...
RUN make restore && \
    make test && \
    CGO_ENABLED=0 GOOS=linux make build-unix

FROM alpine
COPY --from=builder /go/src/bitbucket.hylandqa.net/do/spot/dist/spot /spot
RUN apk add --no-cache --upgrade curl ca-certificates bash && \
    curl -fksSL https://qa-admins.gitlab.hylandqa.net/ca-certificates-hyland/install.sh | bash
ENTRYPOINT ["/spot"]