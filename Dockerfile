# ---------- build stage ----------
FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/fila ./main.go

# ---------- runtime stage ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/fila /app/fila
RUN adduser -D -H appuser
USER appuser
ENTRYPOINT ["/app/fila"]
CMD []