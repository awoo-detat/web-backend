FROM golang:1.22.3 as build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /werewolf

FROM gcr.io/distroless/base-debian11 AS release
WORKDIR /
COPY --from=build /werewolf /werewolf

USER nonroot:nonroot
EXPOSE 43200
CMD ["/werewolf"]
