package order

import (
	"context"

	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
)

type (
	service interface {
		ReserveCheckoutItems(ctx context.Context, request *orderapi.ReserveCheckoutItemsRequest) (*orderapi.ReserveCheckoutItemsResponse, error)
		GetCheckoutReservation(ctx context.Context, request *orderapi.GetCheckoutReservationRequest) (*orderapi.GetCheckoutReservationResponse, error)
		CommitCheckout(ctx context.Context, request *orderapi.CommitCheckoutRequest) (*orderapi.CommitCheckoutResponse, error)
		ReleaseCheckout(ctx context.Context, request *orderapi.ReleaseCheckoutRequest) (*orderapi.ReleaseCheckoutResponse, error)
	}
)
