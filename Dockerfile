FROM golang:1.21-bullseye as build

WORKDIR /go/src/caskadht

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/caskadht ./cmd/caskadht

FROM gcr.io/distroless/static-debian11
COPY --from=build /go/bin/caskadht /usr/bin/

ENTRYPOINT ["/usr/bin/caskadht"]
