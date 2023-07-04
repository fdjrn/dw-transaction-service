package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/db/repository"
	"github.com/fdjrn/dw-transaction-service/internal/handlers/validator"
	"github.com/fdjrn/dw-transaction-service/internal/kafka"
	"github.com/fdjrn/dw-transaction-service/internal/kafka/topic"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"github.com/fdjrn/dw-transaction-service/internal/utilities/str"
	"github.com/gofiber/fiber/v2"
	"time"
)

type TransactionHandler struct {
	repository repository.TransactionRepository
}

func NewTransactionHandler() TransactionHandler {
	return TransactionHandler{repository: repository.NewTransactionRepository()}
}

func (t *TransactionHandler) TopupBalance(c *fiber.Ctx, isMerchant bool) error {

	payload := new(entity.BalanceTransaction)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// validate request
	errMsg, err := validator.ValidateRequest(payload, isMerchant)
	if err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    errMsg,
		})
	}

	if isMerchant {
		payload.TerminalID = ""
		payload.TerminalName = ""
	}

	// 1. validate request
	// 1.1 validate voucher code
	// --- skip dulu 2023-06-26 -----

	// 1.2 check total amount == items amount
	tmpTotalAmt := int64(0)
	for _, item := range payload.Items {
		totAmt := int64(item.Qty) * item.Amount
		tmpTotalAmt += totAmt
	}
	if payload.TotalAmount != tmpTotalAmt {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "total amount mismatch with items amount",
			Data:    nil,
		})
	}

	payload.ReferenceNo = str.GenerateRandomString(8, "", "")
	payload.Status = utilities.TrxStatusPending
	payload.TransType = utilities.TransTopUp
	tStamp := time.Now().UnixMilli()
	payload.CreatedAt = tStamp
	payload.UpdatedAt = tStamp

	t.repository.Model = payload

	// 1.3 check used partnerRefNumber
	if t.repository.IsUsedPartnerRefNumber(payload.PartnerRefNumber) {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "partnerRefNumber already exists",
			Data:    fiber.Map{"partnerRefNumber": payload.PartnerRefNumber},
		})
	}

	// 2. create topup trx data with default value (pending status)
	result, err := t.repository.Create(utilities.TransTopUp)
	if err != nil {
		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// 3. produce message to kafka topic "mdw.transaction.topup.request"
	transData, err := t.repository.FindByID(result)
	if err != nil {
		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	kMsg, _ := json.Marshal(transData)

	err = kafka.ProduceMsg(topic.TopUpRequest, kMsg)
	if err != nil {
		// remove current inserted document
		err = t.repository.RemoveByID(result)
		if err != nil {
			return c.Status(500).JSON(entity.Responses{
				Success: false,
				Message: fmt.Sprintf("failed to remove document with id: %v", result),
				Data:    nil,
			})
		}

		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: "failed to process request",
			Data:    nil,
		})
	}

	// 4. send response http status accepted
	return c.Status(fiber.StatusAccepted).JSON(entity.Responses{
		Success: true,
		Message: "topup request successfully created",
		Data:    transData,
	})

}

func (t *TransactionHandler) DeductBalance(c *fiber.Ctx, isMerchant bool) error {
	return nil
}
