package client

import (
	"context"
	"fmt"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	cartKeyPrefix          = "cart:user:"
	checkoutLockKeyPattern = "cart:user:%d:checkout:lock"
	txRetryLimit           = 5
)

type repository struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) *repository {
	return &repository{
		redis: redisClient,
	}
}

func (r *repository) GetCart(ctx context.Context, userID int64) (*api.Cart, error) {
	payload, err := r.redis.Get(ctx, cartKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, status.Errorf(codes.Internal, "get cart from redis: %v", err)
	}

	cart := &api.Cart{}
	if err := protojson.Unmarshal([]byte(payload), cart); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal cart from redis: %v", err)
	}

	return cart, nil
}

func (r *repository) HasActiveCheckout(ctx context.Context, userID int64) (bool, error) {
	_, err := r.redis.Get(ctx, checkoutLockKey(userID)).Result()
	if err == nil {
		return true, nil
	}
	if err == redis.Nil {
		return false, nil
	}

	return false, status.Errorf(codes.Internal, "get checkout lock from redis: %v", err)
}

func (r *repository) SaveCart(ctx context.Context, cart *api.Cart) error {
	payload, err := protojson.Marshal(cart)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal cart for redis: %v", err)
	}

	if err := r.withCheckoutGuard(ctx, cart.GetUserId(), func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cartKey(cart.GetUserId()), payload, 0)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (r *repository) DeleteCart(ctx context.Context, userID int64) error {
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

func (r *repository) withCheckoutGuard(
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
