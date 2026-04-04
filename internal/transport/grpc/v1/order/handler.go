package order

import (
	"context"

	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
)

type Handler struct {
	service service
	orderapi.UnimplementedCartOrderServiceServer
}

func New(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ReserveCheckoutItems(ctx context.Context, request *orderapi.ReserveCheckoutItemsRequest) (*orderapi.ReserveCheckoutItemsResponse, error) {
	return h.service.ReserveCheckoutItems(ctx, request)
}

func (h *Handler) GetCheckoutReservation(ctx context.Context, request *orderapi.GetCheckoutReservationRequest) (*orderapi.GetCheckoutReservationResponse, error) {
	return h.service.GetCheckoutReservation(ctx, request)
}

func (h *Handler) CommitCheckout(ctx context.Context, request *orderapi.CommitCheckoutRequest) (*orderapi.CommitCheckoutResponse, error) {
	return h.service.CommitCheckout(ctx, request)
}

func (h *Handler) ReleaseCheckout(ctx context.Context, request *orderapi.ReleaseCheckoutRequest) (*orderapi.ReleaseCheckoutResponse, error) {
	return h.service.ReleaseCheckout(ctx, request)
}
