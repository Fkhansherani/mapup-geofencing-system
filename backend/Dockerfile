# Stage 1: compile the Go program
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY . .
# "go mod tidy" downloads the libraries and creates go.sum if it is missing.
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/server .

# Stage 2: tiny runtime image that only contains the compiled program
FROM alpine:3.20
# ca-certificates is needed to connect to cloud databases over TLS.
RUN apk add --no-cache ca-certificates && adduser -D app
COPY --from=build /out/server /server
USER app

EXPOSE 8080
CMD ["/server"]
