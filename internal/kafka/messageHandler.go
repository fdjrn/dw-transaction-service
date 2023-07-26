package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/handlers/consumer"
	"github.com/fdjrn/dw-transaction-service/internal/kafka/topic"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
)

func HandleMessages(message *sarama.ConsumerMessage) {

	var (
		handler = consumer.NewTransactionHandler()
		trx     = new(entity.BalanceTransaction)
		err     error
	)

	utilities.Log.SetPrefix("[CONSUMER] ")

	switch message.Topic {
	case topic.TopUpResult, topic.DeductResult, topic.DistributionResult:
		trx, err = handler.UpdateTransaction(message)
		if err != nil {
			utilities.Log.Println("| failed to process consumed message for topic: ", message.Topic, ", with err: ", err.Error())
			return
		}

		utilities.Log.Printf("| transaction with RefNo: %s, has been successfully updated\n", trx.ReferenceNo)

		// send callback transaction
		err = handler.SendCallback(trx)
		if err != nil {
			utilities.Log.Println(err.Error())

			// change status to TrxStatusCallbackFailed ("07")
			err = handler.HandleCallbackFailed(trx)
			if err != nil {
				utilities.Log.Println("| failed to change transaction status on callback failure: ", err.Error())
				return
			}

			return
		}
		utilities.Log.Printf("| transaction with RefNo: %s, has been successfully send to callback endpoint\n", trx.ReferenceNo)
	case topic.DistributionResultMembers:
		//utilities.Log.Println("| skipped transaction for update")
	default:
		utilities.Log.Println("| unknown topic message")
	}

}
