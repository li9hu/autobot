package executor

import (
	"autobot/internal/database"
	"autobot/internal/models"
	"autobot/internal/notifier"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// LogCleanupCallback 日志清理回调函数类型
type LogCleanupCallback func(taskID uint)

// 全局日志清理回调函数
var logCleanupCallback LogCleanupCallback

// SetLogCleanupCallback 设置日志清理回调函数
func SetLogCleanupCallback(callback LogCleanupCallback) {
	logCleanupCallback = callback
}

// ExecuteTask 执行任务
func ExecuteTask(task *models.Task) {
	startTime := time.Now()

	// 创建任务日志记录
	taskLog := models.TaskLog{
		TaskID:    task.ID,
		StartTime: startTime,
		Status:    "running",
	}

	// 保存日志记录到数据库 - 使用事务和重试机制确保数据一致性
	err := database.WithRetry(func(db *gorm.DB) error {
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		defer tx.Rollback() // 如果没有提交，自动回滚

		if err := tx.Create(&taskLog).Error; err != nil {
			return err
		}

		// 提交初始日志记录
		return tx.Commit().Error
	})

	if err != nil {
		log.Printf("Failed to create task log after retries: %v", err)
		return
	}

	// log.Printf("Starting execution of task: %s (ID: %d)", task.Name, task.ID) // 减少执行开始日志

	// 执行 Python 脚本
	output, errorOutput, err := executePythonScript(task.Script)

	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// 更新日志记录
	taskLog.EndTime = endTime
	taskLog.Duration = duration
	taskLog.Output = output
	taskLog.Error = errorOutput

	// 解析 Python 脚本的 JSON 结果
	var result map[string]interface{}
	if err == nil && output != "" {
		result = parseJSONResult(output)
		if result != nil {
			// 将结果序列化为 JSON 字符串存储
			if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
				taskLog.Result = string(resultJSON)
			}
		}
	}

	if err != nil {
		taskLog.Status = "execution_failed"
		log.Printf("Task execution failed: %s (ID: %d), Error: %v", task.Name, task.ID, err)
	} else if result != nil && result["error"] != nil {
		taskLog.Status = "script_failed"
		log.Printf("Task script error: %s (ID: %d), Error: %v", task.Name, task.ID, result["error"])
	} else {
		taskLog.Status = "success"
		log.Printf("Task execution completed: %s (ID: %d), Duration: %dms", task.Name, task.ID, duration)
	}

	// 保存更新后的日志 - 使用事务和重试机制确保数据一致性
	err = database.WithRetry(func(db *gorm.DB) error {
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		defer tx.Rollback() // 如果没有提交，自动回滚

		if err := tx.Save(&taskLog).Error; err != nil {
			return err
		}

		// 提交更新后的日志记录
		return tx.Commit().Error
	})

	if err != nil {
		log.Printf("Failed to update task log after retries: %v", err)
		return
	}

	// 发送 Bark 通知（如果配置了）
	// 新逻辑：不基于任务状态，而是基于JSON解析和占位符验证
	if task.BarkConfig != "" {
		go sendBarkNotification(task)
	}

	// 调用日志清理回调函数（如果设置了）
	if logCleanupCallback != nil {
		logCleanupCallback(task.ID)
	}
}

// executePythonScript 执行 Python 脚本
func executePythonScript(script string) (output string, errorOutput string, err error) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "autobot_task_")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 创建 Python 脚本文件
	scriptFile := filepath.Join(tempDir, "task_script.py")

	// 确保脚本包含 main() 函数调用
	fullScript := script
	if !containsMainCall(script) {
		fullScript += "\n\nif __name__ == '__main__':\n    main()\n"
		// log.Printf("Added main() call to script") // 减少脚本处理日志
	} else {
		// log.Printf("Script already contains main() call") // 减少脚本处理日志
	}

	// log.Printf("Writing script to file: %s", scriptFile) // 减少文件操作日志
	// log.Printf("Full script content:\n%s", fullScript) // 减少脚本内容日志

	// 同时写入调试文件
	debugFile := filepath.Join(tempDir, "debug.log")
	ioutil.WriteFile(debugFile, []byte(fmt.Sprintf("Script content:\n%s\n", fullScript)), 0644)

	err = ioutil.WriteFile(scriptFile, []byte(fullScript), 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to write script file: %v", err)
	}

	// 执行 Python 脚本（添加 -u 参数强制无缓冲输出）
	cmd := exec.Command("python3", "-u", scriptFile)
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置执行超时（10分钟）
	timeout := time.After(10 * time.Minute)
	done := make(chan error, 1)

	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		output = stdout.String()
		errorOutput = stderr.String()

		// log.Printf("Script execution completed") // 减少执行完成日志
		// log.Printf("Stdout length: %d", len(output)) // 减少输出长度日志
		// log.Printf("Stderr length: %d", len(errorOutput)) // 减少错误长度日志
		// log.Printf("Stdout content: %q", output) // 减少输出内容日志
		// log.Printf("Stderr content: %q", errorOutput) // 减少错误内容日志

		if err != nil {
			return output, errorOutput, fmt.Errorf("script execution failed: %v", err)
		}
		return output, errorOutput, nil

	case <-timeout:
		// 超时，杀死进程
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", "", fmt.Errorf("script execution timeout (10 minutes)")
	}
}

// containsMainCall 检查脚本是否包含 main() 函数调用
func containsMainCall(script string) bool {
	// 只检查是否包含 if __name__ == '__main__': 块
	// 不检查 main() 调用，因为仅定义函数不会执行
	scriptBytes := []byte(script)
	return bytes.Contains(scriptBytes, []byte("if __name__ == '__main__':")) ||
		bytes.Contains(scriptBytes, []byte("if __name__ == \"__main__\":"))
}

// ValidatePythonScript 验证 Python 脚本语法
func ValidatePythonScript(script string) error {
	// 创建临时文件进行语法检查
	tempDir, err := ioutil.TempDir("", "autobot_validate_")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	scriptFile := filepath.Join(tempDir, "validate_script.py")
	err = ioutil.WriteFile(scriptFile, []byte(script), 0644)
	if err != nil {
		return fmt.Errorf("failed to write script file: %v", err)
	}

	// 使用 python -m py_compile 检查语法
	cmd := exec.Command("python3", "-m", "py_compile", scriptFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("syntax error: %s", stderr.String())
	}

	return nil
}

// parseJSONResult 解析 Python 脚本输出的最后一行 JSON
// 如果最后一行是有效的 JSON，返回解析后的 map[string]interface{}
// 否则返回 nil
func parseJSONResult(output string) map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil
	}

	// 获取最后一行
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	if lastLine == "" {
		return nil
	}

	// 首先尝试直接解析为 JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(lastLine), &result); err == nil {
		return result
	}

	// 如果直接解析失败，尝试将 Python 字典格式转换为 JSON 格式
	// 将单引号替换为双引号
	jsonLine := strings.ReplaceAll(lastLine, "'", "\"")
	if err := json.Unmarshal([]byte(jsonLine), &result); err == nil {
		log.Printf("Successfully parsed Python dict format")
		return result
	} else {
		// 都失败了，记录错误并返回 nil
		// log.Printf("Failed to parse result as JSON, original output: %s", lastLine)
		return nil
	}
}

// sendBarkNotification 发送 Bark 通知
func sendBarkNotification(task *models.Task) {
	// 创建通知器实例
	barkNotifier := notifier.New(database.GetDB())

	// 处理并发送 Bark 通知（从最新执行记录中获取 result）
	if err := barkNotifier.ProcessBarkNotification(task); err != nil {
		log.Printf("Failed to send Bark notification for task %d: %v", task.ID, err)
	}
}
