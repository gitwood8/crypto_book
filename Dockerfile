FROM golang:1.23-alpine AS builder


ARG GOBIN=/app
ENV CGO_ENABLED=0
# ENV GO111MODULE=auto
# ENV PATH=$PATH:$GOROOT/bin

WORKDIR /app

COPY . .

RUN apk add git && apk add --no-cache make

RUN go mod download && go mod tidy

RUN make install-tools

RUN make build-service

FROM builder AS dev

RUN apk add --no-cache git make curl

WORKDIR /app

COPY --from=builder /app /app

ENV PATH=/app/bin:$PATH

# EXPOSE 8080

ENTRYPOINT ["make", "watch"]

FROM golang:1.23-alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/bin/wood_post ./wood_post
COPY --from=builder /app/bin/goose ./goose

EXPOSE 8080

RUN addgroup --system norootgroup && adduser --system noroot --ingroup norootgroup
USER noroot

ENTRYPOINT ["./wood_post"]
