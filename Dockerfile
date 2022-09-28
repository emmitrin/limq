FROM golang:1.18-alpine

WORKDIR /limq-api

ADD . /limq-api

RUN go mod download
RUN CGO_ENABLED=0 go build --trimpath -ldflags='-s -w' -o /limq

EXPOSE 8081

CMD ["/limq"]
