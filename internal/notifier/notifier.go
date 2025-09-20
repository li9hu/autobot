package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"autobot/internal/barkhistory"
	"autobot/internal/models"

	"gorm.io/gorm"
)

// 注意：不再提供默认的 Bark API 地址，用户必须配置 Bark 服务器

// Notifier handles Bark notifications
type Notifier struct {
	client         *http.Client
	db             *gorm.DB
	historyManager *barkhistory.BarkHistoryManager
}

// New creates a new Notifier instance
func New(db *gorm.DB) *Notifier {
	return &Notifier{
		client:         &http.Client{},
		db:             db,
		historyManager: barkhistory.NewBarkHistoryManager(),
	}
}

// SendBark sends a Bark notification with the given configuration
// config: map[string]string containing Bark parameters
func (n *Notifier) SendBark(config map[string]string) error {
	// 验证必需的参数
	if config["device_key"] == "" && config["device_keys"] == "" {
		return fmt.Errorf("device_key or device_keys is required for Bark notification")
	}

	// 获取Bark服务器URL
	var barkURL string
	var err error
	if config["device_key"] != "" {
		barkURL, err = n.getBarkServerURL(config["device_key"])
		if err != nil {
			return err
		}
	} else {
		// 对于多设备，使用默认服务器URL
		barkURL, err = n.getDefaultBarkServerURL()
		if err != nil {
			return err
		}
	}

	// 构建请求体
	payload := make(map[string]interface{})

	// 复制所有配置到 payload
	for key, value := range config {
		if value != "" {
			payload[key] = value
		}
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Bark payload: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", barkURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Bark request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// 发送请求
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Bark notification: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark API returned status %d", resp.StatusCode)
	}

	return nil
}

// getBarkServerURL 根据设备密钥获取对应的Bark服务器URL
func (n *Notifier) getBarkServerURL(deviceKey string) (string, error) {
	// 查找设备对应的服务器
	var device models.BarkDevice
	if err := n.db.Preload("Server").Where("device_key = ? AND status = 'active'", deviceKey).First(&device).Error; err == nil {
		if device.Server.URL != "" && device.Server.Status == "active" {
			return device.Server.URL, nil
		}
	}

	// 如果没找到设备配置，尝试获取默认服务器
	var defaultServer models.BarkServer
	if err := n.db.Where("is_default = true AND status = 'active'").First(&defaultServer).Error; err == nil {
		return defaultServer.URL, nil
	}

	// 如果都没找到，返回错误
	return "", fmt.Errorf("no Bark server configuration found for device %s. Please configure a Bark server in the Bark management page", deviceKey)
}

// getDefaultBarkServerURL 获取默认的Bark服务器URL
func (n *Notifier) getDefaultBarkServerURL() (string, error) {
	// 尝试获取默认服务器
	var defaultServer models.BarkServer
	if err := n.db.Where("is_default = true AND status = 'active'").First(&defaultServer).Error; err == nil {
		return defaultServer.URL, nil
	}

	// 如果没找到默认服务器，返回错误
	return "", fmt.Errorf("no default Bark server configuration found. Please configure a default Bark server in the Bark management page")
}

// ProcessBarkNotification processes and sends Bark notification for a task
// task: the task that was executed
func (n *Notifier) ProcessBarkNotification(task *models.Task) error {
	// 从数据库重新获取最新的任务配置，确保配置是最新的
	var latestTask models.Task
	if err := n.db.First(&latestTask, task.ID).Error; err != nil {
		return fmt.Errorf("failed to get latest task config: %v", err)
	}

	// 获取任务的 Bark 配置
	barkConfig, err := latestTask.GetBarkConfig()
	if err != nil {
		return fmt.Errorf("failed to parse Bark config: %v", err)
	}

	// 检查是否有设备配置
	hasDeviceConfig := false
	if len(barkConfig.SelectedDeviceIds) > 0 {
		hasDeviceConfig = true
	} else if barkConfig.DeviceKey != "" {
		hasDeviceConfig = true
	}

	if !hasDeviceConfig {
		log.Printf("No Bark device configuration found for task %d, skipping notification", task.ID)
		return nil
	}

	// 从最新执行记录中获取 result
	result, err := n.getLatestTaskResult(task.ID)
	if err != nil {
		log.Printf("Failed to get result for task %d: %v", task.ID, err)
		return nil
	}

	// 新的过滤逻辑：验证占位符
	if !n.validatePlaceholders(barkConfig, result) {
		log.Printf("Placeholder validation failed for task %d, skipping notification", task.ID)
		return nil
	}

	log.Printf("Placeholder validation passed for task %d, proceeding with notification", task.ID)

	// 根据设备配置发送通知
	if len(barkConfig.SelectedDeviceIds) > 0 {
		// 使用新的设备选择逻辑
		return n.sendBarkToSelectedDevices(barkConfig, result, task.ID)
	} else {
		// 兼容旧的配置方式
		configMap := n.barkConfigToMap(barkConfig)
		finalConfig := n.replacePlaceholders(configMap, result)
		return n.SendBark(finalConfig)
	}
}

// barkConfigToMap converts BarkConfig struct to map[string]string
func (n *Notifier) barkConfigToMap(config *models.BarkConfig) map[string]string {
	configMap := make(map[string]string)

	if config.DeviceKey != "" {
		configMap["device_key"] = config.DeviceKey
	}
	if config.DeviceKeys != "" {
		configMap["device_keys"] = config.DeviceKeys
	}
	if config.Title != "" {
		configMap["title"] = config.Title
	}
	if config.Subtitle != "" {
		configMap["subtitle"] = config.Subtitle
	}
	if config.Body != "" {
		configMap["body"] = config.Body
	}
	if config.Level != "" {
		configMap["level"] = config.Level
	}
	if config.Volume != "" {
		configMap["volume"] = config.Volume
	}
	if config.Badge != "" {
		configMap["badge"] = config.Badge
	}
	if config.Call != "" {
		configMap["call"] = config.Call
	}
	if config.AutoCopy != "" {
		configMap["autoCopy"] = config.AutoCopy
	}
	if config.Copy != "" {
		configMap["copy"] = config.Copy
	}
	if config.Sound != "" {
		configMap["sound"] = config.Sound
	}
	if config.Icon != "" {
		configMap["icon"] = config.Icon
	}
	if config.Group != "" {
		configMap["group"] = config.Group
	}
	if config.Ciphertext != "" {
		configMap["ciphertext"] = config.Ciphertext
	}
	if config.IsArchive != "" {
		configMap["isArchive"] = config.IsArchive
	}
	if config.URL != "" {
		configMap["url"] = config.URL
	}
	if config.Action != "" {
		configMap["action"] = config.Action
	}
	if config.ID != "" {
		configMap["id"] = config.ID
	}
	if config.Delete != "" {
		configMap["delete"] = config.Delete
	}

	return configMap
}

// replacePlaceholders replaces $key placeholders in config values with actual values from result
// Empty placeholders are removed instead of being kept as literal strings
// config: Bark configuration map
// result: JSON result from Python script
func (n *Notifier) replacePlaceholders(config map[string]string, result map[string]interface{}) map[string]string {
	if result == nil {
		result = make(map[string]interface{})
	}

	finalConfig := make(map[string]string)

	// 正则表达式匹配 $key 格式的占位符
	placeholderRegex := regexp.MustCompile(`\$(\w+)`)

	for key, value := range config {
		// 替换占位符
		finalValue := placeholderRegex.ReplaceAllStringFunc(value, func(match string) string {
			// 提取占位符中的 key（去掉 $ 符号）
			placeholder := strings.TrimPrefix(match, "$")

			// 从 result 中查找对应的值
			if resultValue, exists := result[placeholder]; exists {
				// 检查值是否为空
				if resultValue == nil {
					log.Printf("Removing empty placeholder $%s (nil value)", placeholder)
					return "" // 移除空的占位符
				}

				// 检查字符串是否为空
				if strValue, ok := resultValue.(string); ok && strings.TrimSpace(strValue) == "" {
					log.Printf("Removing empty placeholder $%s (empty string)", placeholder)
					return "" // 移除空的占位符
				}

				// 将interface{}转换为字符串
				return n.interfaceToString(resultValue)
			}

			// 如果没有找到对应的值，移除占位符
			log.Printf("Removing undefined placeholder $%s", placeholder)
			return ""
		})

		finalConfig[key] = finalValue
	}

	return finalConfig
}

// interfaceToString 将interface{}类型转换为字符串
func (n *Notifier) interfaceToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case map[string]interface{}, []interface{}:
		// 对于复杂类型，转换为JSON字符串
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getLatestTaskResult 获取任务的最新执行结果
func (n *Notifier) getLatestTaskResult(taskID uint) (map[string]interface{}, error) {
	var taskLog models.TaskLog

	// 查询最新的执行记录（不限制状态）
	err := n.db.Where("task_id = ?", taskID).
		Order("created_at desc").
		First(&taskLog).Error

	if err != nil {
		return nil, fmt.Errorf("no task result found: %v", err)
	}

	result := make(map[string]interface{})

	// 首先尝试解析JSON结果
	jsonParsed := false
	if taskLog.Result != "" {
		var rawResult map[string]interface{}
		if err := json.Unmarshal([]byte(taskLog.Result), &rawResult); err == nil {
			// JSON 解析成功，使用解析后的数据
			for key, value := range rawResult {
				result[key] = value
			}
			jsonParsed = true
			log.Printf("JSON parsed successfully for task %d, status: %s", taskID, taskLog.Status)

			// 对于script_failed状态，JSON中的error字段是有效数据，应该可以作为占位符使用
			if taskLog.Status == "script_failed" {
				log.Printf("Script failed status detected, error field available as placeholder: %v", result["error"])
			}
		} else {
			log.Printf("Failed to parse JSON for task %d: %v", taskID, err)
		}
	}

	// 如果JSON解析失败，不提供默认数据，只使用基础信息
	if !jsonParsed {
		log.Printf("JSON parsing failed for task %d with status %s, no custom data available", taskID, taskLog.Status)
	}

	return result, nil
}

// extractPlaceholders 从配置字符串中提取占位符
func extractPlaceholders(configValue string) []string {
	// 使用正则表达式匹配 $变量名 格式的占位符
	re := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(configValue, -1)

	placeholders := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			placeholder := match[1]
			if !seen[placeholder] {
				placeholders = append(placeholders, placeholder)
				seen[placeholder] = true
			}
		}
	}

	return placeholders
}

// validatePlaceholders 验证占位符，只要有一个占位符有非空值就允许发送
func (n *Notifier) validatePlaceholders(barkConfig *models.BarkConfig, result map[string]interface{}) bool {
	// 收集所有配置字段中的占位符
	allPlaceholders := make(map[string]bool)

	configFields := []string{
		barkConfig.Title,
		barkConfig.Subtitle,
		barkConfig.Body,
		barkConfig.Level,
		barkConfig.Volume,
		barkConfig.Badge,
		barkConfig.Call,
		barkConfig.AutoCopy,
		barkConfig.Copy,
		barkConfig.Sound,
		barkConfig.Icon,
		barkConfig.Group,
		barkConfig.Ciphertext,
		barkConfig.IsArchive,
		barkConfig.URL,
		barkConfig.Action,
		barkConfig.ID,
		barkConfig.Delete,
	}

	// 提取所有占位符
	for _, field := range configFields {
		placeholders := extractPlaceholders(field)
		for _, placeholder := range placeholders {
			allPlaceholders[placeholder] = true
		}
	}

	// 如果没有占位符，直接返回true（允许发送）
	if len(allPlaceholders) == 0 {
		log.Printf("No placeholders found in Bark config, allowing notification")
		return true
	}

	// 检查是否至少有一个占位符有非空值
	validPlaceholders := make([]string, 0)
	invalidPlaceholders := make([]string, 0)

	for placeholder := range allPlaceholders {
		value, exists := result[placeholder]
		if !exists {
			log.Printf("Placeholder $%s not found in result data", placeholder)
			invalidPlaceholders = append(invalidPlaceholders, placeholder)
			continue
		}

		// 检查值是否为空
		if value == nil {
			log.Printf("Placeholder $%s has nil value", placeholder)
			invalidPlaceholders = append(invalidPlaceholders, placeholder)
			continue
		}

		// 检查字符串是否为空
		if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) == "" {
			log.Printf("Placeholder $%s has empty string value", placeholder)
			invalidPlaceholders = append(invalidPlaceholders, placeholder)
			continue
		}

		// 这个占位符有有效值
		validPlaceholders = append(validPlaceholders, placeholder)
	}

	// 只要有至少一个占位符有效，就允许发送
	if len(validPlaceholders) > 0 {
		log.Printf("Placeholder validation passed: %d valid placeholders %v, %d invalid placeholders %v",
			len(validPlaceholders), validPlaceholders, len(invalidPlaceholders), invalidPlaceholders)
		return true
	}

	// 所有占位符都无效
	log.Printf("Placeholder validation failed: all placeholders are invalid %v", invalidPlaceholders)
	return false
}

// sendBarkToSelectedDevices 向选中的设备发送 Bark 通知
func (n *Notifier) sendBarkToSelectedDevices(barkConfig *models.BarkConfig, result map[string]interface{}, taskID uint) error {
	// 获取选中的设备信息
	var devices []models.BarkDevice
	if err := n.db.Where("id IN ? AND status = ?", barkConfig.SelectedDeviceIds, "active").Find(&devices).Error; err != nil {
		return fmt.Errorf("failed to get selected devices: %v", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no active devices found for selected IDs")
	}

	// 准备基础配置
	baseConfig := n.barkConfigToMap(barkConfig)
	baseConfig = n.replacePlaceholders(baseConfig, result)

	var errors []string
	successCount := 0

	// 为每个设备单独发送（以便独立记录和去重）
	for _, device := range devices {
		// 准备设备特定配置
		config := make(map[string]string)
		for k, v := range baseConfig {
			config[k] = v
		}
		config["device_key"] = device.DeviceKey

		// 创建替换后的BarkConfig用于记录
		processedBarkConfig := &models.BarkConfig{
			DeviceKey:  device.DeviceKey,
			Title:      config["title"],
			Subtitle:   config["subtitle"],
			Body:       config["body"], // 这里是替换后的body
			Level:      config["level"],
			Volume:     config["volume"],
			Badge:      config["badge"],
			Call:       config["call"],
			AutoCopy:   config["autoCopy"],
			Copy:       config["copy"],
			Sound:      config["sound"],
			Icon:       config["icon"],
			Group:      config["group"],
			Ciphertext: config["ciphertext"],
			IsArchive:  config["isArchive"],
			URL:        config["url"],
			Action:     config["action"],
			ID:         config["id"],
			Delete:     config["delete"],
		}

		// 创建Bark记录用于去重检查（使用替换后的配置）
		record := models.CreateBarkRecordFromConfig(taskID, device.DeviceKey, processedBarkConfig, "", "", "")

		// 检查是否重复
		isDuplicate, err := n.historyManager.CheckDuplication(record, &barkConfig.Deduplication)
		if err != nil {
			log.Printf("Failed to check duplication for device %s: %v", device.Name, err)
		}

		if isDuplicate {
			log.Printf("Skipping duplicate bark notification for device %s (task %d)", device.Name, taskID)
			// 记录跳过的通知
			record.Status = "skipped"
			record.ErrorMessage = "Duplicate content detected"
			n.historyManager.SaveBarkRecord(record)
			continue
		}

		// 发送通知
		err = n.SendBark(config)
		if err != nil {
			errors = append(errors, fmt.Sprintf("设备 %s: %v", device.Name, err))
			// 记录失败的通知
			record.Status = "failed"
			record.ErrorMessage = err.Error()
		} else {
			successCount++
			log.Printf("Bark notification sent successfully to device: %s", device.Name)
			// 记录成功的通知
			record.Status = "success"
		}

		// 保存记录
		n.historyManager.SaveBarkRecord(record)
	}

	// 处理结果
	if len(errors) > 0 {
		if successCount > 0 {
			log.Printf("Partial success: %d/%d devices, errors: %v", successCount, len(devices), errors)
			return fmt.Errorf("partial failure: %s", strings.Join(errors, "; "))
		} else {
			return fmt.Errorf("all devices failed: %s", strings.Join(errors, "; "))
		}
	}

	return nil
}
