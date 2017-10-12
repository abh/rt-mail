FROM golang:1.9-alpine3.6 AS build

WORKDIR /go/src/github.com/abh/sparkpost-rt
ADD . /go/src/github.com/abh/sparkpost-rt
RUN go-wrapper install

FROM alpine:3.6
RUN apk --no-cache add ca-certificates

RUN addgroup sp && adduser -D -G sp sp

WORKDIR /sp/

COPY --from=build /go/bin/sparkpost-rt /sp/
#COPY --from=build /go/src/github.com/abh/sparkpost-rt/sparkpost-rt.json /etc/sparkpost-rt/config.json

USER sp

ENV CONFIG /etc/sparkpost-rt/config.json

ADD run.sh /sp/

CMD ["/sp/run.sh" ]
