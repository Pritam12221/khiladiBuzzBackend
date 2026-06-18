FROM golang:1.26.1-alpine

WORKDIR /app

COPY  . .

RUN go build -o ./cmd/main ./cmd/main.go

EXPOSE 8080

WORKDIR /app/cmd

CMD [ "./main" ]