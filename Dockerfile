FROM golang:1.22-bookworm AS builder

WORKDIR /go/src/app

COPY ./ .
RUN go build -o build/firefly-iii-bunq-sync .

FROM debian:bookworm-slim
COPY --from=builder /go/src/app/build/firefly-iii-bunq-sync /go/bin/firefly-iii-bunq-sync
ENV PATH="/go/bin:${PATH}"
CMD ["firefly-iii-bunq-sync"]
