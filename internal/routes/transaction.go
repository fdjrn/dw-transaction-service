package routes

import (
	"github.com/fdjrn/dw-transaction-service/internal/handlers"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"github.com/gofiber/fiber/v2"
)

func initTransactionRoutes(router fiber.Router) {
	r := router.Group("/transaction")
	h := handlers.NewTransactionHandler()

	r.Post("/topup", func(c *fiber.Ctx) error {
		return h.CreateTransaction(c, utilities.TransTypeTopUp, false)
	})

	r.Post("/deduct", func(c *fiber.Ctx) error {
		return h.CreateTransaction(c, utilities.TransTypePayment, false)
	})

	// ------------ MERCHANT ------------
	r.Post("/merchant/topup", func(c *fiber.Ctx) error {
		return h.CreateTransaction(c, utilities.TransTypeTopUp, true)
	})

	r.Post("/merchant/deduct", func(c *fiber.Ctx) error {
		return h.CreateTransaction(c, utilities.TransTypePayment, true)
	})

	r.Post("/merchant/distribute", func(c *fiber.Ctx) error {
		//TODO
		return nil
	})

	// ------------ UTILITIES ------------
	r.Post("/inquiry", func(c *fiber.Ctx) error {
		return h.Inquiry(c)
	})

	r.Post("/summary", func(c *fiber.Ctx) error {
		return h.TransactionSummary(c, false)
	})

	r.Post("/summary/period", func(c *fiber.Ctx) error {
		return h.TransactionSummary(c, true)
	})

	r.Post("/summary/topup", func(c *fiber.Ctx) error {
		return h.TransactionSummaryTopup(c, false)
	})

	r.Post("/summary/topup/period", func(c *fiber.Ctx) error {
		return h.TransactionSummaryTopup(c, true)
	})

	r.Post("/summary/deduct", func(c *fiber.Ctx) error {
		return h.TransactionSummaryDeduct(c, false)
	})

	r.Post("/summary/deduct/period", func(c *fiber.Ctx) error {
		return h.TransactionSummaryDeduct(c, true)
	})

}
