FROM golang:1.17.2-alpine3.14 as builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /build/circleci-k8s-agent

FROM alpine:3.14.2

COPY --from=builder /build /usr/local/bin

ENTRYPOINT ["circleci-k8s-agent"]