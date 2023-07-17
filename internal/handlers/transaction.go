package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/fdjrn/dw-transaction-service/internal/db/repository"
	"github.com/fdjrn/dw-transaction-service/internal/handlers/validator"
	"github.com/fdjrn/dw-transaction-service/internal/kafka"
	"github.com/fdjrn/dw-transaction-service/internal/kafka/topic"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"github.com/fdjrn/dw-transaction-service/internal/utilities/str"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type TransactionHandler struct {
	repository repository.TransactionRepository
}

func NewTransactionHandler() TransactionHandler {
	return TransactionHandler{repository: repository.NewTransactionRepository()}
}

func (t *TransactionHandler) rollbackTransaction(c *fiber.Ctx, id interface{}, transType int) error {
	// remove current inserted document
	err := t.repository.RemoveByID(id, transType)
	if err != nil {
		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: fmt.Sprintf("failed to remove document with id: %v", id),
			Data:    nil,
		})
	}

	return c.Status(500).JSON(entity.Responses{
		Success: false,
		Message: "failed to process request, broker or topic not founds",
		Data:    nil,
	})
}

func (t *TransactionHandler) CreateTransaction(c *fiber.Ctx, transType int, isMerchant bool) error {

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

	// validate transferAmount for transType distribution
	if transType == utilities.TransTypeDistribution && payload.TransferAmount == 0 {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "transferAmount cannot be 0",
			Data:    nil,
		})
	}

	// validate partnerTransDate
	_, err = time.Parse("20060102150405", payload.PartnerTransDate)
	if err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid partnerTransDate format. accepted format are 'YYYYMMDDhhmmss'",
			Data:    nil,
		})
	}

	// 1.2 check total amount == items amount
	if transType == utilities.TransTypeDistribution {
		payload.TotalAmount = payload.TransferAmount
		payload.Items = append(payload.Items, entity.TransactionItem{
			//ID:     "",
			//Code:   "",
			Name:   "Merchant Balance distribution",
			Amount: payload.TransferAmount,
			//Price:  0,
			//Qty:    0,
		})
	} else {
		// transType topup & payment only
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
	}

	switch transType {
	case utilities.TransTypeTopUp:
		payload.ReferenceNo = str.GenerateRandomString(10, "", "")
	case utilities.TransTypePayment:
		payload.ReferenceNo = str.GenerateRandomString(8, time.Now().Format("200601"), "")
	case utilities.TransTypeDistribution:
		payload.ReferenceNo = str.GenerateRandomString(10, "TRF-", "")
	default:
		payload.ReferenceNo = "-"
	}

	payload.Status = utilities.TrxStatusPending
	payload.TransType = transType
	tStamp := time.Now().UnixMilli()
	payload.CreatedAt = tStamp
	payload.UpdatedAt = tStamp

	t.repository.Model = payload

	// 1.3 check used partnerRefNumber
	if t.repository.IsUsedPartnerRefNumber(payload.PartnerRefNumber, transType) {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "partnerRefNumber already exists",
			Data:    nil,
		})
	}

	// 2. create topup trx data with default value (pending status)
	result, err := t.repository.Create(transType)
	if err != nil {
		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	// 3. produce message to kafka topic "mdw.transaction.topup.request"
	transData, err := t.repository.FindByID(result, transType)
	if err != nil {
		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	transData.(*entity.BalanceTransaction).TransferAmount = 0

	kMsg, _ := json.Marshal(transData)

	var topicMsg, respMsg string
	switch transType {
	case utilities.TransTypeTopUp:
		topicMsg = topic.TopUpRequest
		respMsg = "topup"
	case utilities.TransTypePayment:
		topicMsg = topic.DeductRequest
		respMsg = "payment"
	case utilities.TransTypeDistribution:
		topicMsg = topic.DistributionRequest
		respMsg = "balance distribution"
	default:
		return t.rollbackTransaction(c, result, transType)
	}

	err = kafka.ProduceMsg(topicMsg, kMsg)
	if err != nil {
		return t.rollbackTransaction(c, result, transType)
	}

	// 4. send response http status accepted
	return c.Status(fiber.StatusAccepted).JSON(entity.Responses{
		Success: true,
		Message: fmt.Sprintf("%s request successful", respMsg),
		Data:    transData,
	})

}

func (t *TransactionHandler) Inquiry(c *fiber.Ctx) error {

	payload := new(entity.BalanceTransaction)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	if payload.TransType == 0 {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid transType value",
			Data:    nil,
		})
	}

	if payload.ReferenceNo == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid referenceNo value",
			Data:    nil,
		})
	}

	t.repository.Model = payload
	err := t.repository.FindByRefNo()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: fmt.Sprintf("cannot find transaction with current referenceNo (%s)", payload.ReferenceNo),
				Data:    nil,
			})
		}

		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	return c.Status(200).JSON(entity.Responses{
		Success: true,
		Message: fmt.Sprintf("transaction successfully fetched with status: %s", t.repository.Model.Status),
		Data:    t.repository.Model,
	})
}

func (t *TransactionHandler) TransactionSummary(c *fiber.Ctx, isPeriod bool) error {
	var (
		err     error
		payload = new(entity.BalanceTransaction)
		result  = new(entity.TransactionSummary)
	)

	if err = c.BodyParser(payload); err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	if payload.PartnerID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid partnerId value",
			Data:    nil,
		})
	}

	if payload.MerchantID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid merchantId value",
			Data:    nil,
		})
	}

	if isPeriod {
		payload.Periods.StartDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.Start, "000000"),
			time.Now().Location(),
		)
		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid start periods",
				Data:    nil,
			})
		}

		payload.Periods.EndDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.End, "235959"),
			time.Now().Location(),
		)

		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid end periods",
				Data:    nil,
			})
		}
	}

	payload.Status = utilities.TrxStatusSuccess

	// populate result data.
	result.PartnerID = payload.PartnerID
	result.MerchantID = payload.MerchantID

	t.repository.Model = payload
	t.repository.Model.TransType = utilities.TransTypeTopUp
	summary, err := t.repository.GetTransactionSummary(isPeriod)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(500).JSON(entity.Responses{
				Success: false,
				Message: fmt.Sprintf("err on summarize total credit, with err: %s", err.Error()),
				Data:    nil,
			})

		}
		//summary = 0
	}

	result.TotalCredit = summary

	t.repository.Model.TransType = utilities.TransTypePayment
	summary, err = t.repository.GetTransactionSummary(isPeriod)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(500).JSON(entity.Responses{
				Success: false,
				Message: fmt.Sprintf("err on summarize total debit, with err: %s", err.Error()),
				Data:    nil,
			})
		}

		//summary = 0
	}
	result.TotalDebit = summary

	return c.Status(200).JSON(entity.Responses{
		Success: true,
		Message: "transaction summary successfully fetched",
		Data:    result,
	})
}

func (t *TransactionHandler) TransactionSummaryTopup(c *fiber.Ctx, isPeriod bool) error {
	var (
		err     error
		payload = new(entity.BalanceTransaction)
		result  = new(entity.TransactionSummaryTopup)
	)

	if err = c.BodyParser(payload); err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	if payload.PartnerID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid partnerId value",
			Data:    nil,
		})
	}

	if payload.MerchantID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid merchantId value",
			Data:    nil,
		})
	}

	if isPeriod {
		payload.Periods.StartDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.Start, "000000"),
			time.Now().Location(),
		)
		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid start periods",
				Data:    nil,
			})
		}

		payload.Periods.EndDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.End, "235959"),
			time.Now().Location(),
		)

		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid end periods",
				Data:    nil,
			})
		}
	}

	utilities.Log.Println("start date: ", payload.Periods.StartDate)
	utilities.Log.Println("end date: ", payload.Periods.EndDate)

	payload.Status = utilities.TrxStatusSuccess

	// populate result data.
	result.PartnerID = payload.PartnerID
	result.MerchantID = payload.MerchantID

	t.repository.Model = payload
	t.repository.Model.TransType = utilities.TransTypeTopUp
	summary, err := t.repository.GetTransactionSummary(isPeriod)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			errMsg := fmt.Sprintf("transaction (topup) not found.")

			if isPeriod {
				errMsg = fmt.Sprintf(
					"transaction (topup) not found for current period %s - %s",
					payload.Periods.Start, payload.Periods.End)
			}

			return c.Status(500).JSON(entity.Responses{
				Success: false,
				Message: errMsg,
				Data:    nil,
			})

		}

		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: fmt.Sprintf("err on summarize total credit, with err: %s", err.Error()),
			Data:    nil,
		})
	}

	result.TotalCredit = summary

	return c.Status(200).JSON(entity.Responses{
		Success: true,
		Message: "topup transaction summary successfully fetched",
		Data:    result,
	})
}

func (t *TransactionHandler) TransactionSummaryDeduct(c *fiber.Ctx, isPeriod bool) error {
	var err error

	payload := new(entity.BalanceTransaction)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: err.Error(),
			Data:    nil,
		})
	}

	if payload.PartnerID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid partnerId value",
			Data:    nil,
		})
	}

	if payload.MerchantID == "" {
		return c.Status(400).JSON(entity.Responses{
			Success: false,
			Message: "invalid merchantId value",
			Data:    nil,
		})
	}

	if isPeriod {
		payload.Periods.StartDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.Start, "000000"),
			time.Now().Location(),
		)
		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid start periods",
				Data:    nil,
			})
		}

		payload.Periods.EndDate, err = time.ParseInLocation(
			"20060102150405",
			fmt.Sprintf("%s%s", payload.Periods.End, "235959"),
			time.Now().Location(),
		)

		if err != nil {
			return c.Status(400).JSON(entity.Responses{
				Success: false,
				Message: "invalid end periods",
				Data:    nil,
			})
		}
	}

	payload.Status = utilities.TrxStatusSuccess

	var result = new(entity.TransactionSummaryDeduct)
	result.PartnerID = payload.PartnerID
	result.MerchantID = payload.MerchantID

	t.repository.Model = payload
	t.repository.Model.TransType = utilities.TransTypePayment
	summary, err := t.repository.GetTransactionSummary(isPeriod)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			errMsg := fmt.Sprintf("transaction (deduct) not found.")

			if isPeriod {
				errMsg = fmt.Sprintf(
					"transaction (deduct) not found for current period %s - %s",
					payload.Periods.Start, payload.Periods.End)
			}

			return c.Status(500).JSON(entity.Responses{
				Success: false,
				Message: errMsg,
				Data:    nil,
			})
		}

		return c.Status(500).JSON(entity.Responses{
			Success: false,
			Message: fmt.Sprintf("err on summarize total debit, with err: %s", err.Error()),
			Data:    nil,
		})
	}

	result.TotalDebit = summary

	return c.Status(200).JSON(entity.Responses{
		Success: true,
		Message: "deduct transaction summary successfully fetched",
		Data:    result,
	})
}
