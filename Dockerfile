FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod ./

RUN go mod tidy

COPY . .

RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]
