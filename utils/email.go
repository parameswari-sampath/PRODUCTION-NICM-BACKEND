package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const ZeptoMailURL = "https://api.zeptomail.in/v1.1/email"

type EmailRecipient struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

type EmailRequest struct {
	From struct {
		Address string `json:"address"`
		Name    string `json:"name,omitempty"`
	} `json:"from"`
	To []struct {
		EmailAddress EmailRecipient `json:"email_address"`
	} `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlbody"`
}

type SendEmailParams struct {
	ToEmail   string
	ToName    string
	Subject   string
	HTMLBody  string
}

type ZeptoMailResponse struct {
	Data []struct {
		Code           string   `json:"code"`
		AdditionalInfo []string `json:"additional_info"`
		Message        string   `json:"message"`
	} `json:"data"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Object    string `json:"object"`
}

// SendEmail sends email via ZeptoMail API and returns the response
func SendEmail(params SendEmailParams) (*ZeptoMailResponse, error) {
	apiKey := os.Getenv("ZEPTO_API_KEY")
	fromEmail := os.Getenv("ZEPTO_FROM_EMAIL")
	fromName := os.Getenv("ZEPTO_FROM_NAME")

	if apiKey == "" || fromEmail == "" {
		return nil, fmt.Errorf("ZeptoMail configuration missing in environment")
	}

	// Construct request body
	emailReq := EmailRequest{
		Subject:  params.Subject,
		HTMLBody: params.HTMLBody,
	}
	emailReq.From.Address = fromEmail
	emailReq.From.Name = fromName
	emailReq.To = []struct {
		EmailAddress EmailRecipient `json:"email_address"`
	}{
		{
			EmailAddress: EmailRecipient{
				Address: params.ToEmail,
				Name:    params.ToName,
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(emailReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", ZeptoMailURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("email send failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse ZeptoMail response
	var zeptoResp ZeptoMailResponse
	if err := json.Unmarshal(body, &zeptoResp); err != nil {
		return nil, fmt.Errorf("failed to parse ZeptoMail response: %w", err)
	}

	return &zeptoResp, nil
}
