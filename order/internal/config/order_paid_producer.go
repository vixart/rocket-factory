package config

import "github.com/IBM/sarama"

type orderPaidProducerConfig struct {
	TopicName string `yaml:"topic_name" env:"ORDER_PAID_TOPIC_NAME" env-default:"order.paid"`
}

func (c *orderPaidProducerConfig) Topic() string {
	return c.TopicName
}

func (c *orderPaidProducerConfig) SaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Producer.Return.Successes = true

	return cfg
}
