package config

type clientConfig struct {
	IAMAddress string `yaml:"iam_address" env:"IAM_ADDRESS" env-default:"localhost:50053"`
}
