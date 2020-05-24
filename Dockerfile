FROM alpine

WORKDIR /app
COPY dist/ ./dist
COPY webterm .
RUN apk add docker-cli

CMD ["./webterm"]
EXPOSE 4567