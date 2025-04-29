FROM golang:1.23

WORKDIR /workspace

COPY . .

RUN (cd /workspace/test/entrypoint && \
 go build -o /usr/local/bin/entrypoint)

RUN (cd /workspace/test && \
 go build -o /usr/local/bin/entrypoint_test)

CMD ["entrypoint_test"]
