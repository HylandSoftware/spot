FROM alpine

ADD ./dist/spot /spot
ENTRYPOINT ["/spot"]