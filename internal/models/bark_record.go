package models

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"time"
)

// BarkRecord Bark发送记录
type BarkRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TaskID         uint      `json:"task_id"`                   // 关联的任务ID
	ContentHash    string    `gorm:"index" json:"content_hash"` // 内容hash，用于去重
	DeviceKey      string    `json:"device_key"`                // 设备密钥
	Title          string    `json:"title"`                     // 通知标题
	Subtitle       string    `json:"subtitle"`                  // 通知副标题
	Body           string    `json:"body"`                      // 通知内容
	Level          string    `json:"level"`                     // 通知级别
	Volume         string    `json:"volume"`                    // 音量
	Badge          string    `json:"badge"`                     // 角标数字
	Call           string    `json:"call"`                      // 通话模式
	AutoCopy       string    `json:"auto_copy"`                 // 自动复制
	Copy           string    `json:"copy"`                      // 复制内容
	Sound          string    `json:"sound"`                     // 通知铃声
	Icon           string    `json:"icon"`                      // 通知图标URL
	Group          string    `json:"group"`                     // 通知分组
	Ciphertext     string    `json:"ciphertext"`                // 加密内容
	IsArchive      string    `json:"is_archive"`                // 是否存档
	URL            string    `json:"url"`                       // 跳转URL
	Action         string    `json:"action"`                    // 自定义动作
	NotificationID string    `json:"notification_id"`           // 通知ID
	Delete         string    `json:"delete"`                    // 删除通知
	Status         string    `json:"status"`                    // 发送状态：success, failed
	ErrorMessage   string    `json:"error_message,omitempty"`   // 错误信息
	ResponseData   string    `json:"response_data,omitempty"`   // 响应数据
	CreatedAt      time.Time `json:"created_at"`
}

// GenerateContentHash 生成内容hash
func (br *BarkRecord) GenerateContentHash() {
	// 构建用于hash的内容结构
	// 注意：包含DeviceKey以确保不同设备的相同内容有不同的hash
	content := struct {
		TaskID     uint   `json:"task_id"`
		DeviceKey  string `json:"device_key"` // 添加设备密钥到hash计算中
		Title      string `json:"title"`
		Subtitle   string `json:"subtitle"`
		Body       string `json:"body"`
		Level      string `json:"level"`
		Volume     string `json:"volume"`
		Badge      string `json:"badge"`
		Call       string `json:"call"`
		AutoCopy   string `json:"auto_copy"`
		Copy       string `json:"copy"`
		Sound      string `json:"sound"`
		Icon       string `json:"icon"`
		Group      string `json:"group"`
		Ciphertext string `json:"ciphertext"`
		IsArchive  string `json:"is_archive"`
		URL        string `json:"url"`
		Action     string `json:"action"`
		Delete     string `json:"delete"`
	}{
		TaskID:     br.TaskID,
		DeviceKey:  br.DeviceKey, // 添加设备密钥到hash计算中
		Title:      br.Title,
		Subtitle:   br.Subtitle,
		Body:       br.Body,
		Level:      br.Level,
		Volume:     br.Volume,
		Badge:      br.Badge,
		Call:       br.Call,
		AutoCopy:   br.AutoCopy,
		Copy:       br.Copy,
		Sound:      br.Sound,
		Icon:       br.Icon,
		Group:      br.Group,
		Ciphertext: br.Ciphertext,
		IsArchive:  br.IsArchive,
		URL:        br.URL,
		Action:     br.Action,
		Delete:     br.Delete,
	}

	// 序列化为JSON并计算MD5
	jsonData, _ := json.Marshal(content)
	hash := md5.Sum(jsonData)
	br.ContentHash = hex.EncodeToString(hash[:])
}

// CreateBarkRecordFromConfig 从BarkConfig创建BarkRecord
func CreateBarkRecordFromConfig(taskID uint, deviceKey string, config *BarkConfig, status string, errorMsg string, responseData string) *BarkRecord {
	record := &BarkRecord{
		TaskID:         taskID,
		DeviceKey:      deviceKey,
		Title:          config.Title,
		Subtitle:       config.Subtitle,
		Body:           config.Body,
		Level:          config.Level,
		Volume:         config.Volume,
		Badge:          config.Badge,
		Call:           config.Call,
		AutoCopy:       config.AutoCopy,
		Copy:           config.Copy,
		Sound:          config.Sound,
		Icon:           config.Icon,
		Group:          config.Group,
		Ciphertext:     config.Ciphertext,
		IsArchive:      config.IsArchive,
		URL:            config.URL,
		Action:         config.Action,
		NotificationID: config.ID,
		Delete:         config.Delete,
		Status:         status,
		ErrorMessage:   errorMsg,
		ResponseData:   responseData,
		CreatedAt:      time.Now(),
	}

	// 生成内容hash
	record.GenerateContentHash()

	return record
}
