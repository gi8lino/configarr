FROM tinygo/tinygo:0.39.0 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/configarr/main.go .

RUN tinygo build -o configarr -opt=s -no-debug main.go

FROM scratch
COPY --from=builder /app/configarr .
ENTRYPOINT ["/configarr"]
