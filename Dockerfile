FROM golang:1.21-alpine

WORKDIR /app

# 1. Copy BOTH mod and sum files first
# (Make sure you have run 'go mod tidy' locally on your machine first!)
COPY go.mod go.sum ./

# 2. Download dependencies based on those files
RUN go mod download

# 3. NOW copy the source code
COPY . .

# 4. Build the application
RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]
