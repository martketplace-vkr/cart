package client

import (
	"context"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
)

type (
	service interface {
		GetCart(ctx context.Context, request *api.GetCartRequest) (*api.GetCartResponse, error)
		AddItem(ctx context.Context, request *api.AddItemRequest) (*api.AddItemResponse, error)
		UpdateItemQuantity(ctx context.Context, request *api.UpdateItemQuantityRequest) (*api.UpdateItemQuantityResponse, error)
		RemoveItem(ctx context.Context, request *api.RemoveItemRequest) (*api.RemoveItemResponse, error)
		ClearCart(ctx context.Context, request *api.ClearCartRequest) (*api.ClearCartResponse, error)
	}
)
