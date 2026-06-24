package config

type clientConfig struct {
	InventoryAddress string `yaml:"inventory_address" env:"INVENTORY_ADDRESS" env-default:"localhost:50051"`
	PaymentAddress   string `yaml:"payment_address" env:"PAYMENT_ADDRESS" env-default:"localhost:50052"`
	IAMAddress       string `yaml:"iam_address" env:"IAM_ADDRESS" env-default:"localhost:50053"`
}
