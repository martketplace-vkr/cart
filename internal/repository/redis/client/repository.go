package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/martketplace-vkr/cart/domain"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	cartKeyPrefix          = "cart:user:"
	checkoutLockKeyPattern = "cart:user:%d:checkout:lock"
	txRetryLimit           = 5
)

type Repository struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) *Repository {
	return &Repository{
		redis: redisClient,
	}
}

func (r *Repository) GetCart(ctx context.Context, userID int64) (*domain.Cart, error) {
	payload, err := r.redis.Get(ctx, cartKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, status.Errorf(codes.Internal, "get cart from redis: %v", err)
	}

	cart := &domain.Cart{}
	if err := json.Unmarshal([]byte(payload), cart); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal cart from redis: %v", err)
	}

	return cart, nil
}

func (r *Repository) HasActiveCheckout(ctx context.Context, userID int64) (bool, error) {
	_, err := r.redis.Get(ctx, checkoutLockKey(userID)).Result()
	if err == nil {
		return true, nil
	}
	if err == redis.Nil {
		return false, nil
	}

	return false, status.Errorf(codes.Internal, "get checkout lock from redis: %v", err)
}

func (r *Repository) SaveCart(ctx context.Context, cart *domain.Cart) error {
	payload, err := json.Marshal(cart)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal cart for redis: %v", err)
	}

	if err := r.withCheckoutGuard(ctx, cart.UserId, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cartKey(cart.UserId), payload, 0)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteCart(ctx context.Context, userID int64) error {
	if err := r.withCheckoutGuard(ctx, userID, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, cartKey(userID))

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func cartKey(userID int64) string {
	return fmt.Sprintf("%s%d", cartKeyPrefix, userID)
}

func checkoutLockKey(userID int64) string {
	return fmt.Sprintf(checkoutLockKeyPattern, userID)
}

func (r *Repository) withCheckoutGuard(
	ctx context.Context,
	userID int64,
	fn func(pipe redis.Pipeliner) error,
) error {
	lockKey := checkoutLockKey(userID)

	for attempt := 0; attempt < txRetryLimit; attempt++ {
		err := r.redis.Watch(ctx, func(tx *redis.Tx) error {
			active, err := tx.Exists(ctx, lockKey).Result()
			if err != nil {
				return status.Errorf(codes.Internal, "check checkout lock in redis: %v", err)
			}
			if active > 0 {
				return status.Error(codes.FailedPrecondition, "checkout in progress")
			}

			_, err = tx.TxPipelined(ctx, fn)
			if err != nil {
				return status.Errorf(codes.Internal, "save cart to redis: %v", err)
			}

			return nil
		}, lockKey)
		if err == redis.TxFailedErr {
			continue
		}
		if err != nil {
			return err
		}

		return nil
	}

	return status.Error(codes.Aborted, "concurrent cart update, please retry")
}
