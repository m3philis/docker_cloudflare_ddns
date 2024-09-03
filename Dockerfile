FROM alpine:latest

RUN apk add bind flarectl curl go

ADD ddns_updater.go /cf-ddns.go

RUN go mod init cf-ddns && go get golang.org/x/exp/slices && go get gopkg.in/yaml.v3

CMD ["go", "run", "/cf-ddns.go"]
