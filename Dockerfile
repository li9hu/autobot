FROM python:latest

WORKDIR /opt

# 拷贝本地 autobot 文件夹和数据库到容器
COPY autobot /opt/autobot
COPY web /opt/web

# 安装依赖
RUN pip3 install --no-cache-dir requests bs4

# 启动命令：后台运行 autobot 并保持容器持续运行
CMD ["sh", "-c", "nohup /opt/autobot >> /opt/run.log 2>&1 & tail -f /dev/null"]
