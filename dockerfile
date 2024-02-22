FROM golang:1.21-alpine

# RUN chown -R trainee:mercari /path/to/db
RUN apk add --no-cache gcc musl-dev
# これ以降はこのディレクトリで操作が行われる(dockerコンテナ内の作業ディレクトリ)
WORKDIR /root

# コピー元のファイルはDockerfile が置いてあるディレクトリ以下の階層に存在する必要あり
COPY go /root/go
COPY db /root/db
# COPY go/app /root/go/app
# COPY go/images /root/go/images
# COPY go/app/main.go /root/go/app


# USER root
RUN cd go && go mod download
WORKDIR /root/go

#RUN addgroup -S mercari && adduser -S trainee -G mercari
#RUN chown -R trainee:mercari /root/db
#RUN chown -R trainee:mercari /root/go
#USER trainee
CMD ["go", "run", "app/main.go"]
