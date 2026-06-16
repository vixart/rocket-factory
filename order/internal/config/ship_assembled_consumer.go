package config

import "github.com/IBM/sarama"

type shipAssembledConsumerConfig struct {
	TopicName string `yaml:"topic_name" env:"SHIP_ASSEMBLED_TOPIC_NAME" env-default:"assembly.ship-assembled"`
	Group     string `yaml:"group" env:"SHIP_ASSEMBLED_CONSUMER_GROUP_ID" env-default:"order-service-ship-assembled-consumer"`
}

func (c *shipAssembledConsumerConfig) Topic() string {
	return c.TopicName
}

func (c *shipAssembledConsumerConfig) GroupID() string {
	return c.Group
}

func (c *shipAssembledConsumerConfig) SaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	return cfg
}
