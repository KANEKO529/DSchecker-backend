// internal/model/clerk_webhook.go
package model

import "encoding/json"

type ClerkWebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type ClerkUserPayload struct {
	ID                    string              `json:"id"`
	Username              string              `json:"username"`
	FirstName             string              `json:"first_name"`
	LastName              string              `json:"last_name"`
	PrimaryEmailAddressID string              `json:"primary_email_address_id"`
	EmailAddresses        []ClerkEmailAddress `json:"email_addresses"`
}

type ClerkEmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

type ClerkDeletedUserPayload struct {
	ID string `json:"id"`
}
