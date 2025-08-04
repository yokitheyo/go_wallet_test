FROM golang:1.23-alpine

RUN apk add --no-cache postgresql-client

WORKDIR /app

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

ENV PATH=/go/bin:$PATH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o wallet-service ./cmd/server

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8080
CMD ["./entrypoint.sh"]
