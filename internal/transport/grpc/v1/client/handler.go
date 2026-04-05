package client

import (
	"context"

	serviceClient "github.com/martketplace-vkr/cart/internal/service/client"
	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service *serviceClient.Service
	client.UnimplementedCartClientServiceServer
}

func New(service *serviceClient.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetCart(ctx context.Context, request *client.GetCartRequest) (resp *client.GetCartResponse, err error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cart, err := h.service.GetCart(ctx, request)
	if err != nil {
		return resp, err
	}

	resp = &client.GetCartResponse{
		Cart: cart.ToProto(),
	}

	return resp, nil
}

func (h *Handler) AddItem(ctx context.Context, request *client.AddItemRequest) (resp *client.AddItemResponse, err error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if request.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	if request.GetVendorId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "vendor_id is required")
	}

	if request.GetQuantity() == 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than zero")
	}

	cart, err := h.service.AddItem(ctx, request)
	if err != nil {
		return resp, err
	}

	resp = &client.AddItemResponse{
		Cart: cart.ToProto(),
	}

	return resp, nil
}

func (h *Handler) UpdateItemQuantity(ctx context.Context, request *client.UpdateItemQuantityRequest) (resp *client.UpdateItemQuantityResponse, err error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if request.GetQuantity() == 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than zero")
	}

	cart, err := h.service.UpdateItemQuantity(ctx, request)
	if err != nil {
		return resp, err
	}

	resp = &client.UpdateItemQuantityResponse{
		Cart: cart.ToProto(),
	}

	return resp, err
}

func (h *Handler) RemoveItem(ctx context.Context, request *client.RemoveItemRequest) (resp *client.RemoveItemResponse, err error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if request.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	cart, err := h.service.RemoveItem(ctx, request)
	if err != nil {
		return nil, err
	}

	resp = &client.RemoveItemResponse{
		Cart: cart.ToProto(),
	}

	return resp, err
}

func (h *Handler) ClearCart(ctx context.Context, request *client.ClearCartRequest) (resp *client.ClearCartResponse, err error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cart, err := h.service.ClearCart(ctx, request)
	if err != nil {
		return resp, err
	}

	resp = &client.ClearCartResponse{
		Cart: cart.ToProto(),
	}

	return resp, err
}
