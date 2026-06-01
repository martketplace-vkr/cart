package order

import (
	"context"

	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
)

type (
	repository interface {
		ReserveCheckoutItems(
			ctx context.Context,
			userID int64,
			checkoutID string,
			productIDs []int64,
			expectedCartVersion uint64,
			preferredCurrencyID int64,
		) (*orderapi.CheckoutReservation, error)
		GetCheckoutReservation(ctx context.Context, userID int64, checkoutID string) (*orderapi.CheckoutReservation, error)
		CommitCheckout(ctx context.Context, userID int64, checkoutID string) (*orderapi.CommitCheckoutResponse, error)
		ReleaseCheckout(ctx context.Context, userID int64, checkoutID string, reason string) (*orderapi.ReleaseCheckoutResponse, error)
	}
)
