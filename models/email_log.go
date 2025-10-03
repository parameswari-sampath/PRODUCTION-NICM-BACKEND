package models

import (
	"time"
)

type EmailLog struct {
	ID              int       `json:"id"`
	StudentID       *int      `json:"student_id"`
	Email           string    `json:"email"`
	Subject         string    `json:"subject"`
	Status          string    `json:"status"`
	RequestID       *string   `json:"request_id"`
	ResponseCode    *string   `json:"response_code"`
	ResponseMessage *string   `json:"response_message"`
	ZeptoResponse   *string   `json:"zepto_response"`
	ErrorMessage    *string   `json:"error_message"`
	SentAt          time.Time `json:"sent_at"`
}
