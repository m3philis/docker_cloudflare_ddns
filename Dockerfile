FROM alpine:latest

RUN apk add bind flarectl curl go

ADD ddns_updater.go /cf-ddns.go

CMD ["go run /cf-ddns.go"]
