package client

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/big"
	"slices"
	"strings"
	"time"

	clientapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	checkoutReservationKeyPattern = "cart:user:%d:checkout:%s"
	checkoutReservationTTL        = 5 * time.Minute
)

func (r *Repository) ReserveCheckoutItems(
	ctx context.Context,
	userID int64,
	checkoutID string,
	productIDs []int64,
	expectedCartVersion uint64,
) (*orderapi.CheckoutReservation, error) {
	lockKey := checkoutLockKey(userID)
	reservationKey := checkoutReservationKey(userID, checkoutID)
	cKey := cartKey(userID)

	for attempt := 0; attempt < txRetryLimit; attempt++ {
		var reservation *orderapi.CheckoutReservation

		err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
			lockValue, err := tx.Get(ctx, lockKey).Result()
			switch {
			case err == nil:
				if lockValue != checkoutID {
					return status.Error(codes.FailedPrecondition, "another checkout is already in progress")
				}

				reservation, err = getReservationTx(ctx, tx, reservationKey)
				if err != nil {
					return err
				}
				return nil
			case err != redis.Nil:
				return status.Errorf(codes.Internal, "get checkout lock from redis: %v", err)
			}

			cart, err := getCartTx(ctx, tx, cKey)
			if err != nil {
				return err
			}
			if cart == nil || len(cart.GetItems()) == 0 {
				return status.Error(codes.NotFound, "cart not found")
			}

			currentVersion, err := calculateCartVersion(cart)
			if err != nil {
				return err
			}
			if expectedCartVersion != 0 && expectedCartVersion != currentVersion {
				return status.Error(codes.Aborted, "cart version mismatch")
			}

			reservation, err = buildReservation(cart, userID, checkoutID, productIDs, currentVersion)
			if err != nil {
				return err
			}

			payload, err := protojson.Marshal(reservation)
			if err != nil {
				return status.Errorf(codes.Internal, "marshal checkout reservation for redis: %v", err)
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, lockKey, checkoutID, checkoutReservationTTL)
				pipe.Set(ctx, reservationKey, payload, checkoutReservationTTL)

				return nil
			})
			if err != nil {
				return status.Errorf(codes.Internal, "save checkout reservation to redis: %v", err)
			}

			return nil
		}, cKey, lockKey, reservationKey)
		if err == redis.TxFailedErr {
			continue
		}
		if err != nil {
			return nil, err
		}

		return reservation, nil
	}

	return nil, status.Error(codes.Aborted, "concurrent cart update, please retry")
}

func (r *Repository) GetCheckoutReservation(
	ctx context.Context,
	userID int64,
	checkoutID string,
) (*orderapi.CheckoutReservation, error) {
	reservation, err := getReservation(ctx, r.redis, checkoutReservationKey(userID, checkoutID))
	if err != nil {
		return nil, err
	}
	if reservation == nil {
		return nil, status.Error(codes.NotFound, "checkout reservation not found")
	}

	return reservation, nil
}

func (r *Repository) CommitCheckout(
	ctx context.Context,
	userID int64,
	checkoutID string,
) (*orderapi.CommitCheckoutResponse, error) {
	lockKey := checkoutLockKey(userID)
	reservationKey := checkoutReservationKey(userID, checkoutID)
	cKey := cartKey(userID)

	for attempt := 0; attempt < txRetryLimit; attempt++ {
		var response *orderapi.CommitCheckoutResponse

		err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
			lockValue, err := tx.Get(ctx, lockKey).Result()
			switch {
			case err == redis.Nil:
				return status.Error(codes.NotFound, "checkout reservation not found")
			case err != nil:
				return status.Errorf(codes.Internal, "get checkout lock from redis: %v", err)
			case lockValue != checkoutID:
				return status.Error(codes.FailedPrecondition, "checkout is owned by another request")
			}

			reservation, err := getReservationTx(ctx, tx, reservationKey)
			if err != nil {
				return err
			}
			if reservation == nil {
				return status.Error(codes.NotFound, "checkout reservation not found")
			}

			cart, err := getCartTx(ctx, tx, cKey)
			if err != nil {
				return err
			}

			removedProductIDs := reservationProductIDs(reservation)
			cartVersion := uint64(0)
			var cartPayload []byte
			shouldDeleteCart := true

			if cart != nil {
				cart.Items = filterOutReservedItems(cart.GetItems(), removedProductIDs)
				if len(cart.GetItems()) > 0 {
					prepareCartForPersist(cart)

					cartVersion, err = calculateCartVersion(cart)
					if err != nil {
						return err
					}

					cartPayload, err = marshalProtoCartPayload(cart)
					if err != nil {
						return status.Errorf(codes.Internal, "marshal cart for redis: %v", err)
					}
					shouldDeleteCart = false
				}
			}

			now := timestamppb.Now()
			response = &orderapi.CommitCheckoutResponse{
				UserId:            userID,
				CheckoutId:        checkoutID,
				RemovedProductIds: removedProductIDs,
				CartVersion:       cartVersion,
				CommittedAt:       now,
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				if shouldDeleteCart {
					pipe.Del(ctx, cKey)
				} else {
					pipe.Set(ctx, cKey, cartPayload, 0)
				}
				pipe.Del(ctx, lockKey, reservationKey)

				return nil
			})
			if err != nil {
				return status.Errorf(codes.Internal, "commit checkout in redis: %v", err)
			}

			return nil
		}, cKey, lockKey, reservationKey)
		if err == redis.TxFailedErr {
			continue
		}
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, status.Error(codes.Aborted, "concurrent cart update, please retry")
}

func (r *Repository) ReleaseCheckout(
	ctx context.Context,
	userID int64,
	checkoutID string,
	_ string,
) (*orderapi.ReleaseCheckoutResponse, error) {
	lockKey := checkoutLockKey(userID)
	reservationKey := checkoutReservationKey(userID, checkoutID)
	cKey := cartKey(userID)

	for attempt := 0; attempt < txRetryLimit; attempt++ {
		var response *orderapi.ReleaseCheckoutResponse

		err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
			lockValue, err := tx.Get(ctx, lockKey).Result()
			switch {
			case err == nil && lockValue != checkoutID:
				return status.Error(codes.FailedPrecondition, "checkout is owned by another request")
			case err != nil && err != redis.Nil:
				return status.Errorf(codes.Internal, "get checkout lock from redis: %v", err)
			}

			reservation, err := getReservationTx(ctx, tx, reservationKey)
			if err != nil {
				return err
			}

			cartVersion := uint64(0)
			cart, err := getCartTx(ctx, tx, cKey)
			if err != nil {
				return err
			}
			if cart != nil {
				cartVersion, err = calculateCartVersion(cart)
				if err != nil {
					return err
				}
			}

			response = &orderapi.ReleaseCheckoutResponse{
				UserId:             userID,
				CheckoutId:         checkoutID,
				ReleasedProductIds: reservationProductIDs(reservation),
				CartVersion:        cartVersion,
				ReleasedAt:         timestamppb.Now(),
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, lockKey, reservationKey)

				return nil
			})
			if err != nil {
				return status.Errorf(codes.Internal, "release checkout in redis: %v", err)
			}

			return nil
		}, cKey, lockKey, reservationKey)
		if err == redis.TxFailedErr {
			continue
		}
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, status.Error(codes.Aborted, "concurrent cart update, please retry")
}

func buildReservation(
	cart *clientapi.Cart,
	userID int64,
	checkoutID string,
	productIDs []int64,
	cartVersion uint64,
) (*orderapi.CheckoutReservation, error) {
	requested := make(map[int64]struct{}, len(productIDs))
	for _, productID := range productIDs {
		requested[productID] = struct{}{}
	}

	items := make([]*orderapi.CheckoutCartItem, 0, len(requested))
	found := make(map[int64]struct{}, len(requested))

	for _, item := range cart.GetItems() {
		if item == nil {
			continue
		}

		if _, ok := requested[item.GetProductId()]; !ok {
			continue
		}

		if !item.GetAvailable() {
			return nil, status.Errorf(codes.FailedPrecondition, "product %d is unavailable", item.GetProductId())
		}
		if item.GetAvailableQuantity() != 0 && item.GetQuantity() > item.GetAvailableQuantity() {
			return nil, status.Errorf(codes.FailedPrecondition, "product %d exceeds available quantity", item.GetProductId())
		}

		totalPrice := item.GetTotalPrice()
		if totalPrice == "" {
			totalPrice = multiplyDecimalString(item.GetUnitPrice(), item.GetQuantity())
		}

		items = append(items, &orderapi.CheckoutCartItem{
			ProductId:   item.GetProductId(),
			VendorId:    item.GetVendorId(),
			VendorName:  item.GetVendorName(),
			ProductName: item.GetProductName(),
			ImageUrl:    item.GetImageUrl(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   normalizeDecimalString(item.GetUnitPrice()),
			TotalPrice:  totalPrice,
		})
		found[item.GetProductId()] = struct{}{}
	}

	for _, productID := range productIDs {
		if _, ok := found[productID]; !ok {
			return nil, status.Errorf(codes.NotFound, "product %d not found in cart", productID)
		}
	}

	now := timestamppb.Now()

	return &orderapi.CheckoutReservation{
		CheckoutId:    checkoutID,
		UserId:        userID,
		Items:         items,
		Totals:        buildReservationTotals(items),
		CartVersion:   cartVersion,
		ReservedUntil: timestamppb.New(now.AsTime().Add(checkoutReservationTTL)),
		CreatedAt:     now,
	}, nil
}

func buildReservationTotals(items []*orderapi.CheckoutCartItem) *orderapi.CheckoutCartTotals {
	var totalItems uint32
	subtotal := big.NewRat(0, 1)

	for _, item := range items {
		if item == nil {
			continue
		}

		totalItems += item.GetQuantity()
		subtotal.Add(subtotal, parseDecimal(item.GetTotalPrice()))
	}

	return &orderapi.CheckoutCartTotals{
		TotalItems: totalItems,
		Subtotal:   decimalToString(subtotal),
		Discount:   "0",
		Total:      decimalToString(subtotal),
	}
}

func reservationProductIDs(reservation *orderapi.CheckoutReservation) []int64 {
	if reservation == nil {
		return nil
	}

	productIDs := make([]int64, 0, len(reservation.GetItems()))
	for _, item := range reservation.GetItems() {
		if item == nil {
			continue
		}

		productIDs = append(productIDs, item.GetProductId())
	}

	return productIDs
}

func filterOutReservedItems(items []*clientapi.CartItem, productIDs []int64) []*clientapi.CartItem {
	if len(items) == 0 || len(productIDs) == 0 {
		return items
	}

	filtered := make([]*clientapi.CartItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if slices.Contains(productIDs, item.GetProductId()) {
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func prepareCartForPersist(cart *clientapi.Cart) {
	if cart.GetId() == 0 {
		cart.Id = cart.GetUserId()
	}
	if cart.GetCreatedAt() == nil {
		cart.CreatedAt = timestamppb.Now()
	}

	recalculateCart(cart)
	cart.UpdatedAt = timestamppb.Now()
}

func recalculateCart(cart *clientapi.Cart) {
	if cart.Totals == nil {
		cart.Totals = zeroTotals()
	}
	if cart.Items == nil {
		cart.Items = []*clientapi.CartItem{}
	}

	var totalItems uint32
	subtotal := big.NewRat(0, 1)
	discount := big.NewRat(0, 1)

	for _, item := range cart.Items {
		if item == nil {
			continue
		}

		item.UnitPrice = normalizeDecimalString(item.GetUnitPrice())
		item.TotalPrice = multiplyDecimalString(item.GetUnitPrice(), item.GetQuantity())
		totalItems += item.GetQuantity()

		if item.GetSelected() {
			subtotal.Add(subtotal, parseDecimal(item.GetTotalPrice()))
		}
	}

	total := new(big.Rat).Sub(subtotal, discount)
	if total.Sign() < 0 {
		total = big.NewRat(0, 1)
	}

	cart.Totals.TotalItems = totalItems
	cart.Totals.Subtotal = decimalToString(subtotal)
	cart.Totals.Discount = decimalToString(discount)
	cart.Totals.Total = decimalToString(total)
}

func zeroTotals() *clientapi.CartTotals {
	return &clientapi.CartTotals{
		TotalItems: 0,
		Subtotal:   "0",
		Discount:   "0",
		Total:      "0",
	}
}

func multiplyDecimalString(value string, quantity uint32) string {
	base := parseDecimal(value)
	multiplier := big.NewRat(int64(quantity), 1)

	return decimalToString(new(big.Rat).Mul(base, multiplier))
}

func normalizeDecimalString(value string) string {
	return decimalToString(parseDecimal(value))
}

func parseDecimal(value string) *big.Rat {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return big.NewRat(0, 1)
	}

	rat, ok := new(big.Rat).SetString(trimmed)
	if ok {
		return rat
	}

	return big.NewRat(0, 1)
}

func decimalToString(value *big.Rat) string {
	if value == nil {
		return "0"
	}

	if value.IsInt() {
		return value.Num().String()
	}

	result := value.FloatString(2)
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")
	if result == "" || result == "-0" {
		return "0"
	}

	return result
}

func calculateCartVersion(cart *clientapi.Cart) (uint64, error) {
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(cart)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "marshal cart for version: %v", err)
	}

	hash := fnv.New64a()
	if _, err := hash.Write(payload); err != nil {
		return 0, status.Errorf(codes.Internal, "hash cart version: %v", err)
	}

	return hash.Sum64(), nil
}

func getCartTx(ctx context.Context, tx *redis.Tx, key string) (*clientapi.Cart, error) {
	payload, err := tx.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get cart from redis: %v", err)
	}

	cart, err := unmarshalProtoCartPayload(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal cart from redis: %v", err)
	}

	return cart, nil
}

func getReservation(ctx context.Context, getter redis.Cmdable, key string) (*orderapi.CheckoutReservation, error) {
	payload, err := getter.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get checkout reservation from redis: %v", err)
	}

	reservation := &orderapi.CheckoutReservation{}
	if err := protojson.Unmarshal([]byte(payload), reservation); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal checkout reservation from redis: %v", err)
	}

	return reservation, nil
}

func getReservationTx(ctx context.Context, tx *redis.Tx, key string) (*orderapi.CheckoutReservation, error) {
	return getReservation(ctx, tx, key)
}

func checkoutReservationKey(userID int64, checkoutID string) string {
	return fmt.Sprintf(checkoutReservationKeyPattern, userID, checkoutID)
}
