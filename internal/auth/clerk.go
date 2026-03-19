package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
)

type ClerkVerifier struct {
	jwksClient *jwks.Client
	jwkStore   JWKStore
}

func NewClerkVerifier() (*ClerkVerifier, error) {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("CLERK_SECRET_KEY is not set")
	}

	clerk.SetKey(secretKey)

	config := &clerk.ClientConfig{}
	config.Key = clerk.String(secretKey)

	return &ClerkVerifier{
		jwksClient: jwks.NewClient(config),
		jwkStore:   NewJWKStore(),
	}, nil
}

// GetSessionTokenFromRequest:
// 1. Authorization: Bearer xxx
// 2. __session cookie
func GetSessionTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(authHeader, prefix) {
			return strings.TrimPrefix(authHeader, prefix)
		}
	}

	cookie, err := r.Cookie("__session")
	if err == nil {
		return cookie.Value
	}

	return ""
}

func (v *ClerkVerifier) VerifySessionToken(
	ctx context.Context,
	sessionToken string,
) (*clerk.SessionClaims, error) {
	if sessionToken == "" {
		return nil, errors.New("session token is empty")
	}

	// まずキャッシュ済み JWK を試す
	jwk := v.jwkStore.GetJWK()
	if jwk != nil {
		claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{
			Token: sessionToken,
			JWK:   jwk,
		})
		if err == nil {
			return claims, nil
		}

		// 鍵ローテーションなどで失敗したらキャッシュ破棄
		v.jwkStore.Clear()
	}

	unsafeClaims, err := clerkjwt.Decode(ctx, &clerkjwt.DecodeParams{
		Token: sessionToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	jwk, err = clerkjwt.GetJSONWebKey(ctx, &clerkjwt.GetJSONWebKeyParams{
		KeyID:      unsafeClaims.KeyID,
		JWKSClient: v.jwksClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get json web key: %w", err)
	}

	v.jwkStore.SetJWK(jwk)

	claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{
		Token: sessionToken,
		JWK:   jwk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	return claims, nil
}