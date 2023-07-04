package routes

import (
	"github.com/fdjrn/dw-transaction-service/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func initTransactionRoutes(router fiber.Router) {
	r := router.Group("/transaction")
	h := handlers.NewTransactionHandler()

	r.Post("/topup", func(c *fiber.Ctx) error {
		return h.TopupBalance(c, false)
	})

	r.Post("/deduct", func(c *fiber.Ctx) error {
		return nil
		//return h.Deduct(c, false)
	})

	// ------------ MERCHANT ------------
	r.Post("/merchant/topup", func(c *fiber.Ctx) error {
		return h.TopupBalance(c, true)
	})

	r.Post("/merchant/deduct", func(c *fiber.Ctx) error {
		//return h.Unregister(c)
		return nil
	})

	r.Post("/merchant/distribute", func(c *fiber.Ctx) error {
		//return h.GetMerchants(c)
		return nil
	})

	// ------------ UTILITIES ------------
	r.Post("/inquiry", func(c *fiber.Ctx) error {
		//return h.TransactionInquiry(c)
		return nil
	})

}
