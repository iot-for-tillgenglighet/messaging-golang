FROM golang AS builder

ENV GO111MODULE=on

# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .


WORKDIR /app/test/messaging-golang

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build

FROM alpine

COPY --from=builder /app/test/messaging-golang/messaging-golang /app/

ENTRYPOINT ["/app/messaging-golang"]
