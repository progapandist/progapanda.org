# Standalone image, if you want to run the TUI on its own:
#   docker build -t hello2 . && docker run --rm -it hello2
FROM golang:1.26.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go content.md ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hello2 .

FROM alpine
WORKDIR /app
COPY --from=build /hello2 ./hello2
COPY entrypoint.sh ./entrypoint.sh
COPY canihackit.hack ./canihackit.hack
ENV TERM=xterm-256color
ENTRYPOINT ["./entrypoint.sh"]
CMD ["./hello2"]
