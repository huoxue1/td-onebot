package conf

import (
	"github.com/botuniverse/go-libonebot"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"os"
	path2 "path"
)

var (
	Version = "v0.0.0"
	Impl    = "td-onebot"
)

func GetVersion() string {
	return Version
}

func GetImpl() string {
	return Impl
}

type Config struct {
	libonebot.Config
	Auth struct {
		Type      string `json:"type" yaml:"type"   mapstructure:"type" toml:"type"`
		ApiId     int    `json:"api_id" yaml:"api_id" mapstructure:"api_id" toml:"api_id"`
		ApiHash   string `json:"api_hash" yaml:"api_hash" mapstructure:"api_hash" toml:"api_hash"`
		BotToken  string `json:"bot_token" yaml:"bot_token" mapstructure:"bot_token" toml:"bot_token"`
		LoginType string `json:"login_type" yaml:"login_type" mapstructure:"login_type"`
	} `json:"auth" mapstructure:"auth" yaml:"auth" toml:"auth"`

	Cache struct {
		CacheType string `json:"cache_type" yaml:"cache_type" mapstructure:"cache_type"`
		CacheDir  string `json:"cache_dir" yaml:"cache_dir" mapstructure:"cache_dir"`
		NutsDB    struct {
			Dir string `json:"dir" yaml:"dir" mapstructure:"dir"`
		} `json:"nuts_db" yaml:"nuts_db" mapstructure:"nuts_db"`
		Redis struct {
			Host     string `json:"host" yaml:"host" mapstructure:"host"`
			Port     int    `json:"port" yaml:"port" mapstructure:"port"`
			Password string `json:"password" yaml:"password" mapstructure:"password"`
			Db       int    `json:"db" yaml:"db" mapstructure:"db"`
		} `yaml:"redis" mapstructure:"redis"`
	} `json:"cache" yaml:"cache" mapstructure:"cache"`

	Proxy string `json:"proxy" yaml:"proxy" mapstructure:"proxy"`
}

var (
	config *Config
)

func InitConfig(path string) {
	if path == "" {
		path = "./config/"
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Errorln("配置文件不存在")
			data, err := yaml.Marshal(new(Config))
			if err != nil {
				log.Errorln(err.Error())
			}
			err = os.WriteFile(path2.Join(path, "config.yaml"), data, 0666)
			if err != nil {
				log.Errorln(err.Error())
			}
			os.Exit(3)
		} else {
			log.Errorln("解析配置文件未知错误" + err.Error())
			os.Exit(3)
		}
	}
	config = new(Config)
	err := viper.Unmarshal(config)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	_ = os.MkdirAll(config.Cache.CacheDir, 0644)
}

func GetConfig() *Config {
	return config
}
