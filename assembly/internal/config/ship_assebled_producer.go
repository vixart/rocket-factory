package config

import "github.com/IBM/sarama"

type shipAssembledProducerConfig struct {
	TopicName string `yaml:"topic_name" env:"SHIP_ASSEMBLED_TOPIC_NAME" env-default:"assembly.ship-assembled"`
}

func (c *shipAssembledProducerConfig) Topic() string {
	return c.TopicName
}

func (c *shipAssembledProducerConfig) SaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Producer.Return.Successes = true

	return cfg
}
