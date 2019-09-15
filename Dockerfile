FROM golang:1.13-alpine3.10 AS build

WORKDIR /go/src/github.com/abh/rt-mail
ADD . /go/src/github.com/abh/rt-mail
RUN go install

FROM alpine:3.10
RUN apk --no-cache add ca-certificates

RUN addgroup rt-mail && adduser -D -G rt-mail rt-mail

WORKDIR /rt-mail/

COPY --from=build /go/bin/rt-mail /rt-mail/
#COPY --from=build /go/src/github.com/abh/sparkpost-rt/sparkpost-rt.json /etc/sparkpost-rt/config.json

USER rt-mail

ENV CONFIG /etc/rt-mail/config.json

ADD run.sh /rt-mail/

CMD ["/rt-mail/run.sh" ]
