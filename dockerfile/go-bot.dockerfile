FROM golang:1.22-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/kkk-go-bot ./cmd/kkk-go-bot

FROM alpine:3.20

RUN apk add --no-cache ca-certificates iproute2 procps
COPY --from=build /out/kkk-go-bot /usr/local/bin/kkk-go-bot
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/kkk-go-bot"]
