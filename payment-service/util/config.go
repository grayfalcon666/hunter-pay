package util

import (
	"github.com/spf13/viper"
)

type Config struct {
	AlipayAppID       string `mapstructure:"APP_ID"`
	AlipayPrivateKey  string `mapstructure:"APP_PRIVATE_KEY"`
	AlipayPublicKey   string `mapstructure:"ALIPAY_PUBLIC_KEY"`
	TokenSymmetricKey string `mapstructure:"TOKEN_SYMMETRIC_KEY"`

	RabbitMQURL       string `mapstructure:"RABBITMQ_URL"`
	HTTPServerAddress string `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress string `mapstructure:"GRPC_SERVER_ADDRESS"`

	WebhookBaseURL    string `mapstructure:"WEBHOOK_BASE_URL"`
	FrontedBaseURL    string `mapstructure:"FRONTEND_BASE_URL"`
	SimpleBankAddress string `mapstructure:"SIMPLE_BANK_ADDRESS"`
	GatewayURL        string `mapstructure:"GATEWAY_URL"`
	DBSource          string `mapstructure:"DB_SOURCE"`

	PlatformEscrowAccountID int64  `mapstructure:"PLATFORM_ESCROW_ACCOUNT_ID"`
	EscrowSystemUsername    string `mapstructure:"ESCROW_SYSTEM_USERNAME"`
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
