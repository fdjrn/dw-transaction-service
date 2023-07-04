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
		//TODO
		return nil
	})

	// ------------ MERCHANT ------------
	r.Post("/merchant/topup", func(c *fiber.Ctx) error {
		return h.TopupBalance(c, true)
	})

	r.Post("/merchant/deduct", func(c *fiber.Ctx) error {
		//TODO
		return nil
	})

	r.Post("/merchant/distribute", func(c *fiber.Ctx) error {
		//TODO
		return nil
	})

	// ------------ UTILITIES ------------
	r.Get("/inquiry/:refNo", func(c *fiber.Ctx) error {
		//TODO
		return h.Inquiry(c)
	})

}
