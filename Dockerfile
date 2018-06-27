FROM golang:alpine

WORKDIR /go/src/github.com/patrickdevivo/traefik-packet
COPY . .

RUN apk update; apk add curl; apk add git;
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
# RUN dep ensure -vendor-only -v

RUN go build -o cmd/traefik-packet

ENTRYPOINT [ "cmd/traefik-packet" ]