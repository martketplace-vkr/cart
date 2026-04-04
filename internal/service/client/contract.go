package client

import (
	"context"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
)

type (
	repository interface {
		GetCart(ctx context.Context, userID int64) (*api.Cart, error)
		HasActiveCheckout(ctx context.Context, userID int64) (bool, error)
		SaveCart(ctx context.Context, cart *api.Cart) error
		DeleteCart(ctx context.Context, userID int64) error
	}
)
