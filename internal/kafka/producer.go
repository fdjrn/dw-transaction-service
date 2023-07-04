package kafka

import (
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/fdjrn/dw-transaction-service/configs"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"github.com/fdjrn/dw-transaction-service/internal/utilities/str"

	"strings"
)

var (
	Producer sarama.SyncProducer
	//SaramaLogger = log.New(os.Stdout, "[PRODUCER] ", log.LstdFlags)
)

func initProducer() error {
	splitBrokers := strings.Split(configs.MainConfig.Kafka.Brokers, ",")

	conf := configs.NewSaramaConfig()
	conf.Producer.Retry.Max = configs.MainConfig.Kafka.Producer.RetryMax
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true
	//conf.Producer.Partitioner = sarama.NewRandomPartitioner
	conf.Producer.Idempotent = configs.MainConfig.Kafka.Producer.Idempotent

	syncProducer, err := sarama.NewSyncProducer(splitBrokers, conf)
	if err != nil {
		return errors.New(fmt.Sprintf("| failed to create producer: %s", err.Error()))
	}

	Producer = syncProducer
	utilities.Log.Println("| kafka client (producer) >> created")

	return nil
}

func Initialize() error {
	switch configs.MainConfig.Kafka.Mode {
	case "producer":
		if err := initProducer(); err != nil {
			return err
		}
	case "consumer":
		return nil
	}

	return nil
}

func ProduceMsg(topic string, payload []byte) error {
	utilities.Log.SetPrefix("[PRODUCER] ")
	partition, offset, err := Producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(str.GetUnixTime()),
		Value: sarama.StringEncoder(payload),
	})
	if err != nil {
		utilities.Log.Println("| failed to send message to ", topic, err)
		return err
	}

	utilities.Log.Printf("| message successfully wrote at partition: %d, offset: %d\n", partition, offset)
	return nil
}
