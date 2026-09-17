# Imagen base oficial de Go
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar go.mod y go.sum si existe (usando wildcard)
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Compilar el binario de Go
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Imagen final ligera para ejecución
FROM alpine:latest
WORKDIR /root/

# Copiar el ejecutable y las carpetas necesarias
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/data ./data

# Puerto por defecto
EXPOSE 8080

CMD ["./main"]