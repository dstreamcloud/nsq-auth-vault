FROM golang:1.14 AS build

WORKDIR /go/src/github.com/dstream.cloud/nsq-auth-vault
ADD . /go/src/github.com/dstream.cloud/nsq-auth-vault

RUN go build -o /go/bin/nsq-auth-vault cmd/nsq-auth-vault

FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/nsq-auth-vault /
CMD ["/nsq-auth-vault"]
