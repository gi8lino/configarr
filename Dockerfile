FROM tinygo/tinygo:0.44.0 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/configarr/*.go ./cmd/configarr/
COPY internal/ ./internal/

RUN tinygo build -o configarr -opt=s -no-debug ./cmd/configarr

FROM scratch
COPY --from=builder /app/configarr .
ENTRYPOINT ["/configarr"]
