FROM golang:1.21-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o coffee-machine .

FROM scratch

COPY --from=build /src/coffee-machine /coffee-machine

EXPOSE 8080

ENTRYPOINT ["/coffee-machine"]
