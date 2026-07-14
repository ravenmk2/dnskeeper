FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /dnskeeper ./cmd/dnskeeper

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /dnskeeper /dnskeeper
COPY config.example.toml /config.toml
EXPOSE 8080
ENTRYPOINT ["/dnskeeper", "-config", "/config.toml"]
