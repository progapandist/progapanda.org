FROM alpine
RUN apk add docker-cli

WORKDIR /app
COPY dist/ ./dist
COPY webterm .

CMD ["./webterm"]
EXPOSE 4567