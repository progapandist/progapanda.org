FROM ubuntu

WORKDIR /app
COPY bin/hello .

ENV LANG=C.UTF-8

CMD ["./hello"]
