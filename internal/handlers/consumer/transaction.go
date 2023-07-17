package consumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/fdjrn/dw-transaction-service/configs"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/db/repository"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"github.com/fdjrn/dw-transaction-service/internal/utilities/crypt"
	"github.com/gofiber/fiber/v2"
)

type TransactionHandler struct {
	transactionRepository repository.TransactionRepository
}

func NewTransactionHandler() TransactionHandler {
	return TransactionHandler{
		transactionRepository: repository.NewTransactionRepository(),
	}
}

func (t *TransactionHandler) SendCallback(trx *entity.BalanceTransaction) error {

	utilities.Log.Printf("| sending payload transaction (%s) to callback api...\n", trx.ReferenceNo)
	payload, _ := json.Marshal(trx)
	signatr := crypt.CreateNewHMAC(
		configs.MainConfig.ExternalResource.CallBackAPI.MDLSecret,
		payload,
	)

	a := fiber.AcquireAgent()
	req := a.Request()
	req.SetRequestURI(configs.MainConfig.ExternalResource.CallBackAPI.MDLTransaction)
	req.Header.SetMethod(fiber.MethodPost)
	req.Header.SetContentType("application/json")
	req.Header.Add("X-Webhook-Signature", signatr)
	req.SetBody(payload)

	if err := a.Parse(); err != nil {
		return errors.New(fmt.Sprintf("error on send transaction callback with refNo: %s, %s", trx.ReferenceNo, err.Error()))
	}

	callbackResponse := new(entity.CallBackResponseAPI)
	code, body, _ := a.Bytes()
	err := json.Unmarshal(body, callbackResponse)
	if err != nil {
		return errors.New("error on marshaling response body")
	}

	errMsg := ""

	if code != 200 {
		errMsg = fmt.Sprintf("error on send transaction callback with refNo: %s, %s", trx.ReferenceNo, callbackResponse.Message)
		return errors.New(errMsg)
	}

	if !callbackResponse.Success {
		errMsg = fmt.Sprintf("unsuccessful callback response message: %s", callbackResponse.Message)
		return errors.New(errMsg)
	}

	return nil
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
		return nil, err
	}

	return data, nil

}

func (t *TransactionHandler) HandleCallbackFailed(data *entity.BalanceTransaction) error {

	data.Status = utilities.TrxStatusCallbackFailed
	t.transactionRepository.Model = data
	err := t.transactionRepository.Update(data.TransType)
	if err != nil {
		return err
	}

	return nil

}
