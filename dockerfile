FROM golang:1.21-alpine

RUN addgroup -S mercari && adduser -S trainee -G mercari
# RUN chown -R trainee:mercari /path/to/db

# コピー元のファイルはDockerfile が置いてあるディレクトリ以下の階層に存在する必要あり
COPY db /db
COPY go/app /go/app
COPY go/go.mod /go/go.mod
COPY go/go.sum /go/go.sum

RUN go mod tidy

# これ以降はこのディレクトリで操作が行われる(dockerコンテナ内の作業ディレクトリ)
WORKDIR /go/app

RUN chown -R trainee:mercari /go/app

USER trainee

CMD ["go", "run", "main.go"]
```
