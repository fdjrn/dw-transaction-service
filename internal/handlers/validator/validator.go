package validator

import (
	"errors"
	"github.com/fdjrn/dw-transaction-service/internal/db/entity"
	"github.com/gofiber/fiber/v2"
)

func ValidateRequest(payload interface{}, isMerchant bool) (interface{}, error) {
	var msg []string

	switch p := payload.(type) {
	case *entity.BalanceTransaction:
		if p.PartnerTransDate == "" {
			msg = append(msg, "partnerTransDate cannot be empty.")
		}

		if p.PartnerRefNumber == "" {
			msg = append(msg, "partnerRefNumber cannot be empty.")
		}

		if p.PartnerID == "" {
			msg = append(msg, "partnerId cannot be empty.")
		}

		if p.MerchantID == "" {
			msg = append(msg, "merchantId cannot be empty.")
		}

		if !isMerchant && p.TerminalID == "" {
			msg = append(msg, "terminalId cannot be empty.")
		}

		for _, item := range p.Items {
			if item.Name == "" {
				msg = append(msg, "item name cannot be empty.")
			}

			if item.Amount == 0 {
				msg = append(msg, "item amount must be greater than 0.")
			}
		}

	default:
	}

	if len(msg) > 0 {
		return msg, errors.New("request validation status failed")
	}
	return msg, nil

}

func HeaderValidation(c *fiber.Ctx) (string, error) {
	reqHeader := c.GetReqHeaders()

	if v, found := reqHeader["Origin"]; found {
		//log.Println("value for Origin: ", v)
		return v, nil
	}
	return "", errors.New("origin key header not found")
}
