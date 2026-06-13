package config

import "github.com/IBM/sarama"

type orderPaidConsumerConfig struct {
	TopicName string `yaml:"topic_name" env:"ORDER_PAID_TOPIC_NAME" env-default:"order.paid"`
	Group     string `yaml:"group" env:"ORDER_PAID_CONSUMER_GROUP_ID" env-default:"assembly-service-order-paid-consumer"`
}

func (c *orderPaidConsumerConfig) Topic() string {
	return c.TopicName
}

func (c *orderPaidConsumerConfig) GroupID() string {
	return c.Group
}

func (c *orderPaidConsumerConfig) SaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	return cfg
}
