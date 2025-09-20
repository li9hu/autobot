package models

import (
	"encoding/json"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Task 任务模型
type Task struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	Name                string         `json:"name" gorm:"not null"`
	Description         string         `json:"description"`
	Script              string         `json:"script" gorm:"type:text;not null"`
	CronExpr            string         `json:"cron_expr" gorm:"not null"`              // cron 表达式
	Status              string         `json:"status" gorm:"default:inactive"`         // active, inactive
	BarkConfig          string         `json:"bark_config" gorm:"type:text"`           // Bark 通知配置 JSON
	TimeExclusionConfig string         `json:"time_exclusion_config" gorm:"type:text"` // 时间排除配置 JSON
	LastRun             *time.Time     `json:"last_run"`
	NextRun             *time.Time     `json:"next_run"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

// TaskLog 任务执行日志模型
type TaskLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"not null;index"`
	Task      Task      `json:"task" gorm:"foreignKey:TaskID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"` // success, execution_failed, script_failed, running
	Output    string    `json:"output" gorm:"type:text"`
	Error     string    `json:"error" gorm:"type:text"`
	Result    string    `json:"result" gorm:"type:text"` // Python 脚本返回的 JSON 结果
	Duration  int64     `json:"duration"`                // 执行时间（毫秒）
	CreatedAt time.Time `json:"created_at"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	Script              string `json:"script" binding:"required"`
	CronExpr            string `json:"cron_expr" binding:"required"`
	Status              string `json:"status"`
	BarkConfig          string `json:"bark_config"`           // Bark 配置 JSON
	TimeExclusionConfig string `json:"time_exclusion_config"` // 时间排除配置 JSON
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Script              string `json:"script"`
	CronExpr            string `json:"cron_expr"`
	Status              string `json:"status"`
	BarkConfig          string `json:"bark_config"`           // Bark 配置 JSON
	TimeExclusionConfig string `json:"time_exclusion_config"` // 时间排除配置 JSON
}

// BarkServer Bark服务器配置模型
type BarkServer struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`            // 服务器名称
	URL         string         `json:"url" gorm:"not null"`             // 服务器地址
	Description string         `json:"description"`                     // 描述
	IsDefault   bool           `json:"is_default" gorm:"default:false"` // 是否为默认服务器
	Status      string         `json:"status" gorm:"default:active"`    // active, inactive
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// BarkDevice Bark设备配置模型
type BarkDevice struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`       // 设备名称
	DeviceKey   string         `json:"device_key" gorm:"not null"` // 设备密钥
	Description string         `json:"description"`                // 描述
	ServerID    uint           `json:"server_id"`                  // 关联的服务器ID
	Server      BarkServer     `json:"server" gorm:"foreignKey:ServerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	IsDefault   bool           `json:"is_default" gorm:"default:false"` // 是否为默认设备
	Status      string         `json:"status" gorm:"default:active"`    // active, inactive
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// BarkConfig Bark 通知配置结构体
type BarkConfig struct {
	DeviceKey         string `json:"device_key"`          // 设备密钥（兼容旧版本）
	DeviceKeys        string `json:"device_keys"`         // 多设备密钥（逗号分隔，兼容旧版本）
	SelectedDeviceIds []uint `json:"selected_device_ids"` // 选中的设备ID列表
	Title             string `json:"title"`               // 通知标题
	Subtitle          string `json:"subtitle"`            // 通知副标题
	Body              string `json:"body"`                // 通知内容
	Level             string `json:"level"`               // 通知级别：active, timeSensitive, passive
	Volume            string `json:"volume"`              // 音量：0-10
	Badge             string `json:"badge"`               // 角标数字
	Call              string `json:"call"`                // 是否开启通话模式：1开启
	AutoCopy          string `json:"autoCopy"`            // 自动复制：1开启
	Copy              string `json:"copy"`                // 复制内容
	Sound             string `json:"sound"`               // 通知铃声
	Icon              string `json:"icon"`                // 通知图标URL
	Group             string `json:"group"`               // 通知分组
	Ciphertext        string `json:"ciphertext"`          // 加密内容
	IsArchive         string `json:"isArchive"`           // 是否存档：1存档
	URL               string `json:"url"`                 // 点击通知跳转URL
	Action            string `json:"action"`              // 自定义动作
	ID                string `json:"id"`                  // 通知ID
	Delete            string `json:"delete"`              // 删除通知：1删除

	// 去重配置
	Deduplication BarkDeduplicationConfig `json:"deduplication"` // 去重设置
}

// BarkDeduplicationConfig Bark去重配置
type BarkDeduplicationConfig struct {
	Enabled    bool   `json:"enabled"`     // 是否启用去重
	Mode       string `json:"mode"`        // 去重模式：recentN, hash, timeWindow
	RecentN    int    `json:"recent_n"`    // 最近N条记录（recentN模式）
	TimeWindow int    `json:"time_window"` // 时间窗口，单位分钟（timeWindow模式）
}

// UnmarshalJSON 自定义JSON解析，支持字符串类型的数字
func (bdc *BarkDeduplicationConfig) UnmarshalJSON(data []byte) error {
	// 定义临时结构，字段类型更宽泛
	type Alias BarkDeduplicationConfig
	aux := &struct {
		RecentN    interface{} `json:"recent_n"`
		TimeWindow interface{} `json:"time_window"`
		*Alias
	}{
		Alias: (*Alias)(bdc),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 处理 RecentN，支持 string 或 int
	switch v := aux.RecentN.(type) {
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			bdc.RecentN = n
		} else {
			bdc.RecentN = 10 // 默认值
		}
	case float64:
		bdc.RecentN = int(v)
	case int:
		bdc.RecentN = v
	default:
		bdc.RecentN = 10 // 默认值
	}

	// 处理 TimeWindow，支持 string 或 int
	switch v := aux.TimeWindow.(type) {
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			bdc.TimeWindow = n
		} else {
			bdc.TimeWindow = 60 // 默认值
		}
	case float64:
		bdc.TimeWindow = int(v)
	case int:
		bdc.TimeWindow = v
	default:
		bdc.TimeWindow = 60 // 默认值
	}

	return nil
}

// CreateBarkServerRequest 创建Bark服务器请求
type CreateBarkServerRequest struct {
	Name        string `json:"name" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateBarkServerRequest 更新Bark服务器请求
type UpdateBarkServerRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	Status      string `json:"status"`
}

// CreateBarkDeviceRequest 创建Bark设备请求
type CreateBarkDeviceRequest struct {
	Name        string `json:"name" binding:"required"`
	DeviceKey   string `json:"device_key" binding:"required"`
	Description string `json:"description"`
	ServerID    uint   `json:"server_id"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateBarkDeviceRequest 更新Bark设备请求
type UpdateBarkDeviceRequest struct {
	Name        string `json:"name"`
	DeviceKey   string `json:"device_key"`
	Description string `json:"description"`
	ServerID    uint   `json:"server_id"`
	IsDefault   bool   `json:"is_default"`
	Status      string `json:"status"`
}

// GetBarkConfig 解析任务的 Bark 配置
func (t *Task) GetBarkConfig() (*BarkConfig, error) {
	if t.BarkConfig == "" {
		return &BarkConfig{}, nil
	}

	var config BarkConfig
	err := json.Unmarshal([]byte(t.BarkConfig), &config)
	return &config, err
}

// SetBarkConfig 设置任务的 Bark 配置
func (t *Task) SetBarkConfig(config *BarkConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	t.BarkConfig = string(data)
	return nil
}

// TimeExclusionConfig 时间排除配置
type TimeExclusionConfig struct {
	Enabled        bool                `json:"enabled"`         // 是否启用时间排除
	ExclusionRules []TimeExclusionRule `json:"exclusion_rules"` // 排除规则列表
}

// TimeExclusionRule 时间排除规则
type TimeExclusionRule struct {
	Type        string `json:"type"`        // 规则类型：daily, weekly, date_range
	Name        string `json:"name"`        // 规则名称（用于显示）
	StartTime   string `json:"start_time"`  // 开始时间
	EndTime     string `json:"end_time"`    // 结束时间
	Weekdays    []int  `json:"weekdays"`    // 周几（0=周日, 1=周一...6=周六），仅weekly类型使用
	StartDate   string `json:"start_date"`  // 开始日期（YYYY-MM-DD），仅date_range类型使用
	EndDate     string `json:"end_date"`    // 结束日期（YYYY-MM-DD），仅date_range类型使用
	Description string `json:"description"` // 规则描述
}

// GetTimeExclusionConfig 解析任务的时间排除配置
func (t *Task) GetTimeExclusionConfig() (*TimeExclusionConfig, error) {
	if t.TimeExclusionConfig == "" {
		return &TimeExclusionConfig{
			Enabled:        false,
			ExclusionRules: []TimeExclusionRule{},
		}, nil
	}

	var config TimeExclusionConfig
	err := json.Unmarshal([]byte(t.TimeExclusionConfig), &config)
	return &config, err
}

// SetTimeExclusionConfig 设置任务的时间排除配置
func (t *Task) SetTimeExclusionConfig(config *TimeExclusionConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	t.TimeExclusionConfig = string(data)
	return nil
}
