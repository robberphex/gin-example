# golang版本；alpine镜像更小一些
FROM golang:1.16.6-alpine3.14 as builder

# 替换alpine镜像，方便安装构建包
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装构建阶段的依赖
#RUN apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates

# 将代码copy到构建镜像中
# 注意，地址最好不要在GOPATH中
ADD . /workspace

WORKDIR /workspace

# mount构建缓存
# GOPROXY防止下载失败
RUN --mount=type=cache,target=/go \
  env GOPROXY=https://goproxy.cn,direct \
  go build -buildmode=pie -ldflags "-linkmode external -extldflags -static -w" \
  -o /workspace/gin-hello-world

# 运行时镜像
# alpine兼顾了镜像大小和运维性
FROM alpine:3.14

EXPOSE 8080

# 方便运维人员安装需要的包
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 创建日志目录等
# RUN mkdir /var/log/onepilot -p && chmod 777 /var/log/onepilot && touch /var/log/onepilot/.keep

# copy构建产物
COPY --from=builder /workspace/gin-hello-world /app/

# 指定默认的启动命令
CMD ["/app/gin-hello-world"]
