package config

type Config struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
	} `mapstructure:"database"`
}

var GlobalConfig Config

func Init() error {
	// 后续添加配置文件读取逻辑
	GlobalConfig = Config{}
	GlobalConfig.Server.Port = 8080
	GlobalConfig.Server.Host = "0.0.0.0"
	return nil
}
