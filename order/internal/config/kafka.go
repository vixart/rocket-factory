package config

type kafkaConfig struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
}
