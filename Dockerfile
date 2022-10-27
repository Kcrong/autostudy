FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN go build -o main cmd/main.go

WORKDIR /dist

RUN cp /build/main .

FROM scratch

ENV COMMIT_HASH=$GITHUB_SHA

COPY --from=builder /dist/main .

ENTRYPOINT ["/main"]
