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

### Bark 通知配置

1. 在 iOS 设备上安装 [Bark](https://apps.apple.com/cn/app/bark-customed-notifications/id1403753865) 应用
2. 获取设备密钥
3. 在任务详情页面配置 Bark 通知参数

配置示例：
```json
{
  "device_key": "your_device_key_here",
  "title": "$title",
  "body": "$body",
  "group": "$group",
  "sound": "bell.caf"
}
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

## 示例脚本

`examples/` 目录包含了多个示例脚本：

- `simple_task_example.py` - 简单任务示例
- `system_monitor_example.py` - 系统监控示例
- `web_health_check_example.py` - 网站健康检查
- `bark_notification_example.py` - Bark 通知示例
- 更多高级示例...

## 部署说明

详细的部署说明请参考 `DEPLOYMENT.md` 文件。

## 技术栈

- **后端**: Go + Gin + GORM + SQLite
- **前端**: HTML + Tailwind CSS + Alpine.js + JavaScript
- **任务执行**: Python 3
- **通知**: Bark iOS App

## 许可证

MIT License


