FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/payroll-sandbox .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget tzdata
COPY --from=build /out/payroll-sandbox /usr/local/bin/payroll-sandbox
EXPOSE 9001
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:9001/healthz || exit 1
ENTRYPOINT ["payroll-sandbox"]
