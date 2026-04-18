ARG GO_VERSION=1.26.2

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mcp-sqlserver ./cmd

FROM alpine:3.22

RUN addgroup -S mcp && adduser -S mcp -G mcp

COPY --from=build /out/mcp-sqlserver /usr/local/bin/mcp-sqlserver

USER mcp
WORKDIR /home/mcp

ENTRYPOINT ["/usr/local/bin/mcp-sqlserver"]
