package config

import (
	"errors"
	"fmt"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	conf "github.com/ldigit/config"
	"os"
	"path/filepath"
	"strings"
)

// SymlinkMethod
const (
	// Skip 跳过
	Skip string = "skip"
	// Addr 复制链接地址
	Addr string = "addr"
	// File 复制目标文件
	File string = "file"
)

// SyncConfig 同步配置
type SyncConfig struct {
	Local struct {
		Path string `yaml:"path"`
	} `yaml:"local"`
	Remote struct {
		Endpoint  string `yaml:"endpoint"`
		UseSSL    bool   `yaml:"use_ssl"`
		SecretId  string `yaml:"secret_id"`
		SecretKey string `yaml:"secret_key"`
		Bucket    string `yaml:"bucket"`
		Region    string `yaml:"region"`
		Path      string `yaml:"path"`
	} `yaml:"remote"`
	Sync struct {
		RealTime struct {
			Enable   bool `yaml:"enable"`
			HotDelay int  `yaml:"hot_delay"`
		} `yaml:"real_time"`
		CheckJob struct {
			Enable   bool   `yaml:"enable"`
			Interval int    `yaml:"interval"`
			StartAt  string `yaml:"start_at"`
		} `yaml:"check_job"`
		Symlink string   `yaml:"symlink"`
		Ignore  []string `yaml:"ignore,omitempty"`
	} `yaml:"sync"`
	Log []log.OutputConfig `yaml:"log"`
}

// GetConfig 获取解析好的配置
func GetConfig(path string) (*SyncConfig, error) {
	raw := conf.GetGlobalConfig()
	if raw != nil {
		return raw.(*SyncConfig), nil
	}
	cfg := loadConfig(path)
	if cfg == nil {
		return nil, errors.New("configuration is empty, please check the config file path")
	}
	return cfg, nil
}

// GetString 格式化配置成字符串
func (c *SyncConfig) GetString() string {
	s := fmt.Sprintln("****************** ROS *******************")
	s += fmt.Sprintln("Local: -----------------------------------")
	s += fmt.Sprintf("  Path:\t\t| %s\n", c.Local.Path)
	s += fmt.Sprintln("Remote: ----------------------------------")
	s += fmt.Sprintf("  Endpoint:\t| %s\n", c.Remote.Endpoint)
	s += fmt.Sprintf("  SecretId:\t| %s\n", helper.HideSecret(c.Remote.SecretId, 12))
	s += fmt.Sprintf("  SecretKey:\t| %s\n", helper.HideSecret(c.Remote.SecretKey, 12))
	s += fmt.Sprintf("  Bucket:\t| %s\n", c.Remote.Bucket)
	s += fmt.Sprintf("  Region:\t| %s\n", c.Remote.Region)
	s += fmt.Sprintf("  Path:\t\t| %s\n", c.Remote.Path)
	s += fmt.Sprintln("Sync: -----------------------------------")
	s += fmt.Sprintln("  Real-time:")
	s += fmt.Sprintf("    Enable:\t| %t\n", c.Sync.RealTime.Enable)
	s += fmt.Sprintf("    HotDelay:\t| %d minute\n", c.Sync.RealTime.HotDelay)
	s += fmt.Sprintln("  Check-job:")
	s += fmt.Sprintf("    Enable:\t| %t\n", c.Sync.CheckJob.Enable)
	s += fmt.Sprintf("    Interval:\t| %d hour\n", c.Sync.CheckJob.Interval)
	s += fmt.Sprintf("    Start-at:\t| %s\n", c.Sync.CheckJob.StartAt)
	s += fmt.Sprintf("  Symlink:\t| %s\n", c.Sync.Symlink)
	s += fmt.Sprintf("  Ignore:\t| %v\n", c.Sync.Ignore)
	s += fmt.Sprint("******************************************")
	return s
}

func loadConfig(path string) *SyncConfig {
	cfg := &SyncConfig{}
	if err := conf.LoadAndDecode(path, cfg); err != nil {
		return nil
	}

	// 处理local.path为相对路径的情况，替换为绝对路径
	if len(cfg.Local.Path) > 0 && "./" == cfg.Local.Path[0:2] {
		cfg.Local.Path, _ = filepath.Abs(cfg.Local.Path)
	}

	// 处理local.path中带有~的情况，替换为绝对路径
	if len(cfg.Local.Path) > 0 && "~" == cfg.Local.Path[0:1] {
		homeDir, _ := os.UserHomeDir()
		cfg.Local.Path = strings.Replace(cfg.Local.Path, "~", homeDir, 1)
	}

	// 处理remote.path中的前导/
	if len(cfg.Remote.Path) > 0 && "/" == cfg.Remote.Path[0:1] {
		cfg.Remote.Path = strings.TrimLeft(cfg.Remote.Path, "/")
	}

	// 处理Hot delay，最小1分钟，最大60分钟
	if cfg.Sync.RealTime.HotDelay < 1 {
		cfg.Sync.RealTime.HotDelay = 1
	} else if cfg.Sync.RealTime.HotDelay > 60 {
		cfg.Sync.RealTime.HotDelay = 60
	}

	// 处理symlink策略
	cfg.Sync.Symlink = strings.ToLower(cfg.Sync.Symlink)
	if cfg.Sync.Symlink != Skip && cfg.Sync.Symlink != Addr && cfg.Sync.Symlink != File {
		cfg.Sync.Symlink = Skip
	}

	return cfg
}
