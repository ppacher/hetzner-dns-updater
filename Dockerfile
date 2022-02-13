FROM golang:1.17 as build

RUN update-ca-certificates

WORKDIR /go/src/app

COPY go.mod .

ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/hetzner-dns-updater .

FROM gcr.io/distroless/static

COPY --from=build /go/bin/hetzner-dns-updater /go/bin/hetzner-dns-updater

ENTRYPOINT ["/go/bin/hetzner-dns-updater"]