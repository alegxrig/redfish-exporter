FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/redfish-exporter .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/redfish-exporter /usr/local/bin/redfish-exporter
EXPOSE 9812
ENTRYPOINT ["redfish-exporter"]