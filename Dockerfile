FROM golang:1.15 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sq ./cmd/slack-queue/...

FROM alpine:3.12
WORKDIR /src
COPY --from=builder /src/sq /bin/
ENTRYPOINT ["sq"]
