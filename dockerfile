FROM golang:1.21-alpine

RUN apk add --no-cache gcc musl-dev

RUN addgroup -S mercari && adduser -S trainee -G mercari

# dockerコンテナ内の作業ディレクトリ
WORKDIR /root

COPY go /root/go
COPY db /root/db

RUN chown -R trainee:mercari /root/db
RUN chown -R trainee:mercari /root/go
RUN chown -R trainee:mercari /root

WORKDIR /root/go

USER trainee
RUN go mod download
CMD ["go", "run", "app/main.go"]
