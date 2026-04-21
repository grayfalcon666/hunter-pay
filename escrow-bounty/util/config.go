package util

import (
	"github.com/spf13/viper"
)

type Config struct {
	DBSource                  string `mapstructure:"DB_SOURCE"`
	GRPCServerAddress         string `mapstructure:"GRPC_SERVER_ADDRESS"`
	HTTPServerAddress         string `mapstructure:"HTTP_SERVER_ADDRESS"`
	SimpleBankAddress         string `mapstructure:"SIMPLE_BANK_ADDRESS"`
	PlatformEscrowAccountID   int64  `mapstructure:"PLATFORM_ESCROW_ACCOUNT_ID"`
	TokenSymmetricKey         string `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	EscrowSystemUsername      string `mapstructure:"ESCROW_SYSTEM_USERNAME"`
	UserProfileServiceAddress string `mapstructure:"USER_PROFILE_SERVICE_ADDRESS"`
	WSServerAddress          string `mapstructure:"WS_SERVER_ADDRESS"`
	RabbitMQURL              string `mapstructure:"RABBITMQ_URL"`
	UploadPath                string `mapstructure:"UPLOAD_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv() // 自动覆盖 如果系统的环境变量里有同名配置，优先使用系统的

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
