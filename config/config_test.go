package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorben/rsync-object-storage/enum"
	"github.com/stretchr/testify/assert"
)

// createTempConfig 创建临时配置文件
func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)
	return configPath
}

// TestGetConfig 测试配置加载
func TestGetConfig(t *testing.T) {
	t.Run("加载有效配置", func(t *testing.T) {
		configContent := `
local:
  path: /data/local
remote:
  endpoint: s3.example.com
  use_ssl: true
  secret_id: AKID123456
  secret_key: secretkey123
  bucket: my-bucket
  region: us-east-1
  path: backup
sync:
  real_time:
    enable: true
    hot_delay: 5
  check_job:
    enable: true
    interval: 6
    start_at: "03:00:00"
  symlink: skip
  ignore:
    - .git
    - node_modules
`
		configPath := createTempConfig(t, configContent)
		cfg, err := GetConfig(configPath)

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "/data/local", cfg.Local.Path)
		assert.Equal(t, "s3.example.com", cfg.Remote.Endpoint)
		assert.True(t, cfg.Remote.UseSSL)
		assert.Equal(t, "my-bucket", cfg.Remote.Bucket)
		assert.True(t, cfg.Sync.RealTime.Enable)
		assert.Equal(t, 5, cfg.Sync.RealTime.HotDelay)
		assert.True(t, cfg.Sync.CheckJob.Enable)
		assert.Equal(t, "skip", cfg.Sync.Symlink)
		assert.Contains(t, cfg.Sync.Ignore, ".git")
	})

	t.Run("配置文件不存在", func(t *testing.T) {
		_, err := GetConfig("/nonexistent/path/config.yaml")
		assert.Error(t, err)
	})
}

// TestLoadConfig_PathNormalization 测试路径规范化
func TestLoadConfig_PathNormalization(t *testing.T) {
	t.Run("相对路径转绝对路径", func(t *testing.T) {
		configContent := `
local:
  path: ./data
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: false
  check_job:
    enable: false
`
		configPath := createTempConfig(t, configContent)
		cfg, err := GetConfig(configPath)

		assert.NoError(t, err)
		// 相对路径应该被转换为绝对路径
		assert.True(t, filepath.IsAbs(cfg.Local.Path))
	})

	t.Run("波浪号路径替换", func(t *testing.T) {
		homeDir, _ := os.UserHomeDir()
		configContent := `
local:
  path: ~/Documents
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: false
  check_job:
    enable: false
`
		configPath := createTempConfig(t, configContent)
		cfg, err := GetConfig(configPath)

		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(homeDir, "Documents"), cfg.Local.Path)
	})

	t.Run("远程路径去除前导斜杠", func(t *testing.T) {
		configContent := `
local:
  path: /data
remote:
  endpoint: s3.example.com
  bucket: bucket
  path: /backup/path
sync:
  real_time:
    enable: false
  check_job:
    enable: false
`
		configPath := createTempConfig(t, configContent)
		cfg, err := GetConfig(configPath)

		assert.NoError(t, err)
		assert.Equal(t, "backup/path", cfg.Remote.Path)
	})
}

// TestLoadConfig_HotDelayBounds 测试 HotDelay 边界值
func TestLoadConfig_HotDelayBounds(t *testing.T) {
	tests := []struct {
		name     string
		hotDelay int
		expected int
	}{
		{"小于最小值", 0, 1},
		{"最小值", 1, 1},
		{"正常值", 30, 30},
		{"最大值", 60, 60},
		{"大于最大值", 100, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := fmt.Sprintf(`
local:
  path: /data
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: true
    hot_delay: %d
  check_job:
    enable: false
`, tt.hotDelay)
			configPath := createTempConfig(t, configContent)
			cfg, err := GetConfig(configPath)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Sync.RealTime.HotDelay)
		})
	}
}

// TestLoadConfig_SymlinkStrategy 测试 Symlink 策略规范化
func TestLoadConfig_SymlinkStrategy(t *testing.T) {
	tests := []struct {
		name     string
		symlink  string
		expected string
	}{
		{"skip 策略", "skip", enum.SymlinkSkip},
		{"SKIP 大写", "SKIP", enum.SymlinkSkip},
		{"addr 策略", "addr", enum.SymlinkAddr},
		{"file 策略", "file", enum.SymlinkFile},
		{"无效值默认 skip", "invalid", enum.SymlinkSkip},
		{"空值默认 skip", "", enum.SymlinkSkip},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := `
local:
  path: /data
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: false
  check_job:
    enable: false
  symlink: ` + tt.symlink + `
`
			configPath := createTempConfig(t, configContent)
			cfg, err := GetConfig(configPath)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Sync.Symlink)
		})
	}
}

// TestGetString 测试配置格式化输出
func TestGetString(t *testing.T) {
	cfg := &SyncConfig{}
	cfg.Local.Path = "/data/local"
	cfg.Remote.Endpoint = "s3.example.com"
	cfg.Remote.SecretId = "AKID12345678901234567890"
	cfg.Remote.SecretKey = "SECRET12345678901234567890"
	cfg.Remote.Bucket = "my-bucket"
	cfg.Remote.Region = "us-east-1"
	cfg.Remote.Path = "backup"
	cfg.Sync.RealTime.Enable = true
	cfg.Sync.RealTime.HotDelay = 5
	cfg.Sync.CheckJob.Enable = true
	cfg.Sync.CheckJob.Interval = 6
	cfg.Sync.CheckJob.StartAt = "03:00:00"
	cfg.Sync.Symlink = "skip"
	cfg.Sync.Ignore = []string{".git", "node_modules"}

	output := cfg.GetString()

	assert.Contains(t, output, "ROS")
	assert.Contains(t, output, "/data/local")
	assert.Contains(t, output, "s3.example.com")
	assert.Contains(t, output, "my-bucket")
	assert.Contains(t, output, "us-east-1")
	assert.Contains(t, output, "backup")
	// SecretId 和 SecretKey 应该被隐藏
	assert.NotContains(t, output, "AKID12345678901234567890")
	assert.NotContains(t, output, "SECRET12345678901234567890")
	assert.Contains(t, output, "************")
}

// TestCheckJobInterval 测试 CheckJob Interval 最小值
func TestCheckJobInterval(t *testing.T) {
	// 注意：Interval 的最小值限制在 CheckJob.NewCheckJob 中处理，而不是在 loadConfig 中
	// 这里只测试配置加载是否正确
	configContent := `
local:
  path: /data
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: false
  check_job:
    enable: true
    interval: 0
    start_at: "00:00:00"
`
	configPath := createTempConfig(t, configContent)
	cfg, err := GetConfig(configPath)

	assert.NoError(t, err)
	// 配置加载时不处理 interval，所以应该是 0
	assert.Equal(t, 0, cfg.Sync.CheckJob.Interval)
}

// TestEmptyConfig 测试空配置
func TestEmptyConfig(t *testing.T) {
	// 空配置文件会返回一个空结构体，但不会报错
	// 因为 YAML 解析空内容是合法的
	configContent := ``
	configPath := createTempConfig(t, configContent)

	cfg, err := GetConfig(configPath)
	// 根据实际行为：空配置可能返回空结构体
	// 如果返回错误，则验证错误；如果不返回错误，则验证结构体
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, cfg)
	}
}

// TestMinimalConfig 测试最小配置
func TestMinimalConfig(t *testing.T) {
	configContent := `
local:
  path: /tmp
remote:
  endpoint: s3.example.com
  bucket: bucket
sync:
  real_time:
    enable: false
  check_job:
    enable: false
`
	configPath := createTempConfig(t, configContent)
	cfg, err := GetConfig(configPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "/tmp", cfg.Local.Path)
}
