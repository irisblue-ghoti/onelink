package messaging

import (
	"content-service/internal/config"

	"github.com/IBM/sarama"
)

// KafkaClient Kafka客户端
type KafkaClient struct {
	producer sarama.SyncProducer
	topic    string
}

// NewKafkaClient 创建新的Kafka客户端
func NewKafkaClient(config config.KafkaConfig) (*KafkaClient, error) {
	// 创建Kafka配置
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = 5
	kafkaConfig.Producer.Return.Successes = true

	// 创建生产者
	producer, err := sarama.NewSyncProducer(config.Brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaClient{
		producer: producer,
		topic:    config.Topic,
	}, nil
}

// Close 关闭Kafka连接
func (k *KafkaClient) Close() error {
	return k.producer.Close()
}
