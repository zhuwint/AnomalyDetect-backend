FROM golang:latest as builder

ENV GOOS linux
ENV GOPROXY https://goproxy.cn,direct
ENV CGO_ENABLED 0

WORKDIR /root

COPY . .

RUN go mod tidy && go build -o controller ./cmd/controller/main.go

FROM scratch

WORKDIR /root

COPY --from=builder /root/controller .
COPY --from=builder /root/deploy/controller.yaml .

ENV TZ Asia/Shanghai
ENV SERVICE_NAME controller

EXPOSE 8106

CMD ["./controller", "-config", "./controller.yaml"]