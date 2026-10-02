package publisher

import (
	"context"

	"github.com/IBM/sarama"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
}

type KafkaPublisher struct {
	producer sarama.SyncProducer
}

func NewKafkaPublisher(
	brokers []string,
) (*KafkaPublisher, error) {
	config := sarama.NewConfig()

	// The gateway must know whether Kafka accepted the message.
	config.Producer.RequiredAcks = sarama.WaitForAll

	// Return the result of the synchronous publish call.
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &KafkaPublisher{
		producer: producer,
	}, nil
}

func (p *KafkaPublisher) Publish(
	ctx context.Context,
	topic string,
	key []byte,
	value []byte,
) error {
	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	_, _, err := p.producer.SendMessage(message)

	return err
}

func (p *KafkaPublisher) Close() error {
	return p.producer.Close()
}