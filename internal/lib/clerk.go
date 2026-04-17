// internal/lib/clerk.go
// 外部サービスSDKラッパー（認証）
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

func (c *ClerkClient) UpdateName(ctx context.Context, clerkUserID, firstName, lastName string) error {
	params := &user.UpdateParams{
		FirstName: clerk.String(firstName),
		LastName:  clerk.String(lastName),
	}

	_, err := c.userClient.Update(ctx, clerkUserID, params)
	return err
}

// clerk SDKメソッド
func (c *ClerkClient) DeleteUser(ctx context.Context, clerkUserID string) error {
	_, err := c.userClient.Delete(ctx, clerkUserID)
	return err
}
