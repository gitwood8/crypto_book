FROM golang:1.23-alpine AS builder

ENV GOBIN=/app/bin
ENV CGO_ENABLED=0
ENV PATH=$GOBIN:$PATH

WORKDIR /app

COPY go.mod go.sum ./

RUN apk add git && apk add --no-cache make && \
    go mod download && go mod tidy

COPY . .

RUN make install-tools
RUN make build-service

FROM builder AS dev

ENTRYPOINT ["make", "watch"]

FROM golang:1.23-alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/bin/wood_post ./wood_post
COPY --from=builder /app/bin/goose ./goose

RUN addgroup --system norootgroup && adduser --system noroot --ingroup norootgroup
USER noroot

ENTRYPOINT ["./wood_post"]
