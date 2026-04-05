package client

import (
	"context"
	"slices"

	"github.com/martketplace-vkr/cart/domain"
	"github.com/martketplace-vkr/pkg/logger/log"
	"github.com/shopspring/decimal"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetCart(ctx context.Context, request *api.GetCartRequest) (cart *domain.Cart, err error) {
	cart, err = s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		cart = domain.EmptyCart(request.GetUserId())
	} else {
		err = s.catalog.EnrichCartItemByProductInfo(ctx, cart.Items)
		if err != nil {
			log.Errorf("failed enrich cart item (user: %d): %s",request.UserId,  err)
		}
	}

	cart.Recalculate()

	return cart, nil
}

func (s *Service) AddItem(ctx context.Context, request *api.AddItemRequest) (*domain.Cart, error) {
	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart, err := s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		cart = domain.EmptyCart(request.GetUserId())
	}

	index := slices.IndexFunc(cart.Items, func(item domain.CartItem) bool {
		return item.ProductId == request.ProductId
	})

	if index >= 0 {
		cart.Items[index].Quantity += request.GetQuantity()
		if cart.Items[index].VendorId == 0 {
			cart.Items[index].VendorId = request.GetVendorId()
		}
	} else {
		cart.Items = append(cart.Items, domain.CartItem{
			ProductId:         request.GetProductId(),
			VendorId:          request.GetVendorId(),
			Quantity:          request.GetQuantity(),
			AvailableQuantity: request.GetQuantity(),
			Available:         true,
			Selected:          true,
			UnitPrice:         decimal.Zero,
			TotalPrice:        decimal.Zero,
		})
	}

	cart.PrepareForSave()
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *Service) UpdateItemQuantity(ctx context.Context, request *api.UpdateItemQuantityRequest) (*domain.Cart, error) {
	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart, err := s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		return nil, status.Error(codes.NotFound, "cart not found")
	}

	index := slices.IndexFunc(cart.Items, func(item domain.CartItem) bool {
		return item.ProductId == request.ProductId
	})

	if index < 0 {
		return nil, status.Error(codes.NotFound, "cart item not found")
	}

	cart.Items[index].Quantity = request.GetQuantity()
	if cart.Items[index].AvailableQuantity < request.GetQuantity() {
		cart.Items[index].AvailableQuantity = request.GetQuantity()
	}

	cart.PrepareForSave()
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *Service) RemoveItem(ctx context.Context, request *api.RemoveItemRequest) (*domain.Cart, error) {

	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart, err := s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		return nil, status.Error(codes.NotFound, "cart not found")
	}

	index := slices.IndexFunc(cart.Items, func(item domain.CartItem) bool {
		return item.ProductId == request.ProductId
	})

	if index < 0 {
		return nil, status.Error(codes.NotFound, "cart item not found")
	}

	cart.Items = append(cart.Items[:index], cart.Items[index+1:]...)

	if len(cart.Items) == 0 {
		cart = domain.EmptyCart(request.UserId)
		if err := s.repository.DeleteCart(ctx, request.GetUserId()); err != nil {
			return nil, err
		}

		return cart, nil
	}

	cart.PrepareForSave()
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *Service) ClearCart(ctx context.Context, request *api.ClearCartRequest) (*domain.Cart, error) {

	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart := domain.EmptyCart(request.GetUserId())

	if err := s.repository.DeleteCart(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *Service) ensureCartMutable(ctx context.Context, userID int64) error {
	active, err := s.repository.HasActiveCheckout(ctx, userID)
	if err != nil {
		return err
	}

	if active {
		return status.Error(codes.FailedPrecondition, "checkout in progress")
	}

	return nil
}
