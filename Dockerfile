FROM harbor.liasica.com/library/debian:latest

RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources && \
    mkdir /app && \
    apt update && apt install -y bash tzdata ca-certificates && \
    rm -rf /etc/localtime && \
    ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

COPY ./build/release/automatic /app/

WORKDIR /app

ENTRYPOINT ["/app/automatic", "punch", "run", "--config", "/app/configs/config.yaml"]
