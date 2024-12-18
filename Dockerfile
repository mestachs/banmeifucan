# Dockerfile
FROM golang:1.20-alpine
WORKDIR /app
COPY . .
RUN go build -o reverse_proxy .
CMD ["./reverse_proxy"]