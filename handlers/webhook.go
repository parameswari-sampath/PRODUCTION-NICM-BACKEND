package handlers

import (
	"context"
	"log"
	"mcq-exam/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

type WebhookPayload struct {
	EventName []string `json:"event_name"`
	EventMessage []struct {
		RequestID string `json:"request_id"`
		EmailInfo struct {
			Subject string `json:"subject"`
			To []struct {
				EmailAddress struct {
					Address string `json:"address"`
					Name    string `json:"name"`
				} `json:"email_address"`
			} `json:"to"`
		} `json:"email_info"`
		EventData []struct {
			Details []struct {
				Reason            string `json:"reason"`
				BouncedRecipient  string `json:"bounced_recipient"`
				Time              string `json:"time"`
				DiagnosticMessage string `json:"diagnostic_message"`
			} `json:"details"`
		} `json:"event_data"`
	} `json:"event_message"`
}

// ZeptoMailWebhookHandler handles POST /api/webhooks/zeptomail
// Receives bounce notifications from ZeptoMail and updates email status to failed
func ZeptoMailWebhookHandler(c *fiber.Ctx) error {
	var payload WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		// Return 200 even on parse error as per ZeptoMail requirements
		return c.SendStatus(fiber.StatusOK)
	}

	// Process each event message
	for _, msg := range payload.EventMessage {
		if msg.RequestID == "" {
			continue
		}

		// Update email status to failed
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		query := `UPDATE email_logs SET status = 'failed' WHERE request_id = $1`
		_, err := db.Pool.Exec(ctx, query, msg.RequestID)
		cancel()

		if err != nil {
			// Log error but still return 200
			log.Printf("Failed to update email status for request_id %s: %v", msg.RequestID, err)
		}
	}

	// Always return 200 as required by ZeptoMail
	return c.SendStatus(fiber.StatusOK)
}
