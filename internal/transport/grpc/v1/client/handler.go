package client

import (
	"context"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
)

type Handler struct {
	service service
	client.UnimplementedCartClientServiceServer
}

func New(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetCart(ctx context.Context, request *client.GetCartRequest) (*client.GetCartResponse, error) {
	return h.service.GetCart(ctx, request)
}

func (h *Handler) AddItem(ctx context.Context, request *client.AddItemRequest) (*client.AddItemResponse, error) {
	return h.service.AddItem(ctx, request)
}

func (h *Handler) UpdateItemQuantity(ctx context.Context, request *client.UpdateItemQuantityRequest) (*client.UpdateItemQuantityResponse, error) {
	return h.service.UpdateItemQuantity(ctx, request)
}

func (h *Handler) RemoveItem(ctx context.Context, request *client.RemoveItemRequest) (*client.RemoveItemResponse, error) {
	return h.service.RemoveItem(ctx, request)
}

func (h *Handler) ClearCart(ctx context.Context, request *client.ClearCartRequest) (*client.ClearCartResponse, error) {
	return h.service.ClearCart(ctx, request)
}
