package consumer

import (
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/db/repository"
)

type TransactionHandler struct {
	transactionRepository repository.TransactionRepository
}

func NewTransactionHandler() TransactionHandler {
	return TransactionHandler{
		transactionRepository: repository.NewTransactionRepository(),
	}
}

func (t *TransactionHandler) HandleTransactionResult(message *sarama.ConsumerMessage) (*entity.BalanceTransaction, error) {

	data := new(entity.BalanceTransaction)
	err := json.Unmarshal(message.Value, &data)
	if err != nil {
		return nil, err
	}

	// TODO: update transaction status by transaction refNo
	t.transactionRepository.Model = data
	err = t.transactionRepository.Update()
	if err != nil {
		//utilities.Log.Println("| unable to update transaction status, err: ", err.Error())
		return nil, err
	}

	return data, nil

}
