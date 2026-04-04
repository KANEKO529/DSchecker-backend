// internal/lib/clerk.go
package lib

import (
	"context"
	"fmt"
	"net/http"
	"os"

	clerk "github.com/clerk/clerk-sdk-go/v2"
	user "github.com/clerk/clerk-sdk-go/v2/user"
)

type ClerkConfig struct {
	SecretKey  string
	BaseURL    string
	HttpClient *http.Client
}

type ClerkClient struct {
	userClient *user.Client
}

func NewClerkClient(cfg *ClerkConfig) *ClerkClient {
	config := &clerk.ClientConfig{}
	config.Key = clerk.String(cfg.SecretKey)

	return &ClerkClient{
		userClient: user.NewClient(config),
	}
}

func LoadClerkConfig() (*ClerkConfig, error) {
	cfg := &ClerkConfig{
		SecretKey:  os.Getenv("CLERK_SECRET_KEY"),
		BaseURL:    "https://api.clerk.com/v1",
		HttpClient: &http.Client{},
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	return cfg, nil
}

func (c *ClerkClient) UpdateUsername(ctx context.Context, clerkUserID, username string) error {
	params := &user.UpdateParams{
		Username: clerk.String(username),
	}

	_, err := c.userClient.Update(ctx, clerkUserID, params)
	return err
}
