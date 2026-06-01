package order

import (
	"context"

	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *service) ReserveCheckoutItems(
	ctx context.Context,
	request *orderapi.ReserveCheckoutItemsRequest,
) (*orderapi.ReserveCheckoutItemsResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetCheckoutId() == "" {
		return nil, status.Error(codes.InvalidArgument, "checkout_id is required")
	}
	if len(request.GetProductIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_ids are required")
	}
	for _, productID := range request.GetProductIds() {
		if productID == 0 {
			return nil, status.Error(codes.InvalidArgument, "product_ids must be greater than zero")
		}
	}

	reservation, err := s.repository.ReserveCheckoutItems(
		ctx,
		request.GetUserId(),
		request.GetCheckoutId(),
		request.GetProductIds(),
		request.GetExpectedCartVersion(),
		request.GetPreferredCurrencyId(),
	)
	if err != nil {
		return nil, err
	}

	return &orderapi.ReserveCheckoutItemsResponse{
		Reservation: reservation,
	}, nil
}

func (s *service) GetCheckoutReservation(
	ctx context.Context,
	request *orderapi.GetCheckoutReservationRequest,
) (*orderapi.GetCheckoutReservationResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetCheckoutId() == "" {
		return nil, status.Error(codes.InvalidArgument, "checkout_id is required")
	}

	reservation, err := s.repository.GetCheckoutReservation(ctx, request.GetUserId(), request.GetCheckoutId())
	if err != nil {
		return nil, err
	}

	return &orderapi.GetCheckoutReservationResponse{
		Reservation: reservation,
	}, nil
}

func (s *service) CommitCheckout(
	ctx context.Context,
	request *orderapi.CommitCheckoutRequest,
) (*orderapi.CommitCheckoutResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetCheckoutId() == "" {
		return nil, status.Error(codes.InvalidArgument, "checkout_id is required")
	}

	return s.repository.CommitCheckout(ctx, request.GetUserId(), request.GetCheckoutId())
}

func (s *service) ReleaseCheckout(
	ctx context.Context,
	request *orderapi.ReleaseCheckoutRequest,
) (*orderapi.ReleaseCheckoutResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetCheckoutId() == "" {
		return nil, status.Error(codes.InvalidArgument, "checkout_id is required")
	}

	return s.repository.ReleaseCheckout(ctx, request.GetUserId(), request.GetCheckoutId(), request.GetReason())
}
