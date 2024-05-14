FROM golang:1.22.3

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /werewolf

EXPOSE 43200

CMD ["/werewolf"]
