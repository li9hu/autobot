# AutoBot - 自动化任务管理平台

一个基于 Go 和 Web 技术的自动化任务管理平台，支持 Python 脚本执行、定时任务调度和 Bark 通知。

## 功能特性

- 🚀 **任务管理** - 创建、编辑、删除和执行 Python 脚本任务
- ⏰ **定时调度** - 支持 Cron 表达式的定时任务调度
- 📱 **Bark 通知** - 集成 iOS Bark 应用推送通知
- 📊 **执行日志** - 详细的任务执行日志和结果查看
- 🔐 **身份验证** - 基于用户名密码的身份验证系统
- 🎨 **现代界面** - 基于 Tailwind CSS 的响应式 Web 界面

## 快速开始

### 1. 环境要求

- Go 1.19+
- SQLite3

### 2. 编译运行

```bash
# 克隆项目
git clone <repository-url>
cd autoBot

# 安装依赖
go mod tidy

# 编译
go build -o autobot main.go

# 运行
./autobot

或者直接运行 ./start.sh
```

### 3. 访问应用

打开浏览器访问 `http://localhost:8080`

首次访问会跳转到登录页面，可以注册管理员账户。

## 使用说明

### Python 脚本要求

- 脚本必须包含 `main()` 函数
- 如需 Bark 通知，最后一行输出必须是有效的 JSON 格式

示例脚本：
```python
import json
from datetime import datetime

def main():
    print("任务开始执行...")
    
    # 你的任务逻辑
    result = "任务完成"
    
    # 输出 JSON 结果（用于 Bark 通知）
    notification = {
        "title": "任务完成",
        "body": f"执行结果: {result}",
        "group": "自动化任务"
    }
    print(json.dumps(notification, ensure_ascii=False))
```
## 项目结构

```
autoBot/
├── main.go                 # 主程序入口
├── go.mod                  # Go 模块文件
├── start.sh               # 启动脚本
├── example_task.py        # 示例任务脚本
├── examples/              # 更多示例脚本
├── internal/              # 内部模块
│   ├── database/          # 数据库操作
│   ├── executor/          # 任务执行器
│   ├── handlers/          # HTTP 处理器
│   ├── models/            # 数据模型
│   ├── notifier/          # 通知模块
│   └── scheduler/         # 任务调度器
└── web/                   # Web 资源
    ├── static/            # 静态文件
    └── templates/         # HTML 模板
```
<img width="1396" height="576" alt="image" src="https://github.com/user-attachments/assets/ec3de239-bcb7-428a-bb35-a7a2a202a363" />
<img width="1573" height="1166" alt="image" src="https://github.com/user-attachments/assets/828571ba-3f44-4177-9da2-7ec5c7eca521" />




