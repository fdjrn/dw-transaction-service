package consumer

import (
	"encoding/json"
	"errors"
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

	if data == nil {
		return nil, errors.New("empty message value")
	}

	t.transactionRepository.Model = data
	err = t.transactionRepository.Update(data.TransType)
	if err != nil {
		//utilities.Log.Println("| unable to update transaction status, err: ", err.Error())
		return nil, err
	}

	return data, nil

}
