FROM golang:1.20-alpine3.17 AS build

WORKDIR /go/src/github.com/abh/rt-mail
ADD . /go/src/github.com/abh/rt-mail
RUN go install

FROM alpine:3.17
RUN apk --no-cache add ca-certificates

RUN addgroup rt-mail && adduser -D -G rt-mail rt-mail

WORKDIR /rt-mail/

COPY --from=build /go/bin/rt-mail /rt-mail/
COPY --from=build /go/src/github.com/abh/rt-mail/rt-mail.json.sample /etc/rt-mail/config.json.sample

USER rt-mail

ENV CONFIG /etc/rt-mail/config.json

ADD run.sh /rt-mail/

CMD ["/rt-mail/run.sh" ]
