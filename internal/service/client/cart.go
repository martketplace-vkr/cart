package client

import (
	"context"
	"math/big"
	"slices"
	"strings"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) GetCart(ctx context.Context, request *api.GetCartRequest) (*api.GetCartResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cart, err := s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		cart = newCart(request.GetUserId())
	}

	recalculateCart(cart)

	return &api.GetCartResponse{Cart: cart}, nil
}

func (s *service) AddItem(ctx context.Context, request *api.AddItemRequest) (*api.AddItemResponse, error) {
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
	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart, err := s.repository.GetCart(ctx, request.GetUserId())
	if err != nil {
		return nil, err
	}

	if cart == nil {
		cart = newCart(request.GetUserId())
	}

	index := slices.IndexFunc(cart.GetItems(), func(item *api.CartItem) bool {
		return item.GetProductId() == request.GetProductId()
	})

	if index >= 0 {
		cart.Items[index].Quantity += request.GetQuantity()
		if cart.Items[index].VendorId == 0 {
			cart.Items[index].VendorId = request.GetVendorId()
		}
	} else {
		cart.Items = append(cart.Items, &api.CartItem{
			ProductId:         request.GetProductId(),
			VendorId:          request.GetVendorId(),
			Quantity:          request.GetQuantity(),
			AvailableQuantity: request.GetQuantity(),
			Available:         true,
			Selected:          true,
			UnitPrice:         "0",
			TotalPrice:        "0",
		})
	}

	prepareCartForSave(cart)
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return &api.AddItemResponse{Cart: cart}, nil
}

func (s *service) UpdateItemQuantity(ctx context.Context, request *api.UpdateItemQuantityRequest) (*api.UpdateItemQuantityResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if request.GetQuantity() == 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than zero")
	}
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

	index := slices.IndexFunc(cart.GetItems(), func(item *api.CartItem) bool {
		return item.GetProductId() == request.GetProductId()
	})
	if index < 0 {
		return nil, status.Error(codes.NotFound, "cart item not found")
	}

	cart.Items[index].Quantity = request.GetQuantity()
	if cart.Items[index].AvailableQuantity < request.GetQuantity() {
		cart.Items[index].AvailableQuantity = request.GetQuantity()
	}

	prepareCartForSave(cart)
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return &api.UpdateItemQuantityResponse{Cart: cart}, nil
}

func (s *service) RemoveItem(ctx context.Context, request *api.RemoveItemRequest) (*api.RemoveItemResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if request.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
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

	index := slices.IndexFunc(cart.GetItems(), func(item *api.CartItem) bool {
		return item.GetProductId() == request.GetProductId()
	})
	if index < 0 {
		return nil, status.Error(codes.NotFound, "cart item not found")
	}

	cart.Items = append(cart.Items[:index], cart.Items[index+1:]...)

	if len(cart.GetItems()) == 0 {
		clearCart(cart)
		if err := s.repository.DeleteCart(ctx, request.GetUserId()); err != nil {
			return nil, err
		}

		return &api.RemoveItemResponse{Cart: cart}, nil
	}

	prepareCartForSave(cart)
	if err := s.repository.SaveCart(ctx, cart); err != nil {
		return nil, err
	}

	return &api.RemoveItemResponse{Cart: cart}, nil
}

func (s *service) ClearCart(ctx context.Context, request *api.ClearCartRequest) (*api.ClearCartResponse, error) {
	if request.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := s.ensureCartMutable(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	cart := newCart(request.GetUserId())
	clearCart(cart)

	if err := s.repository.DeleteCart(ctx, request.GetUserId()); err != nil {
		return nil, err
	}

	return &api.ClearCartResponse{Cart: cart}, nil
}

func newCart(userID int64) *api.Cart {
	return &api.Cart{
		Id:     userID,
		UserId: userID,
		Items:  []*api.CartItem{},
		Totals: zeroTotals(),
	}
}

func clearCart(cart *api.Cart) {
	createdAt := cart.GetCreatedAt()
	*cart = api.Cart{
		Id:        cart.GetUserId(),
		UserId:    cart.GetUserId(),
		Items:     []*api.CartItem{},
		Totals:    zeroTotals(),
		CreatedAt: createdAt,
		UpdatedAt: timestamppb.Now(),
	}
}

func prepareCartForSave(cart *api.Cart) {
	if cart.GetId() == 0 {
		cart.Id = cart.GetUserId()
	}
	if cart.GetCreatedAt() == nil {
		cart.CreatedAt = timestamppb.Now()
	}

	recalculateCart(cart)
	cart.UpdatedAt = timestamppb.Now()
}

func recalculateCart(cart *api.Cart) {
	if cart.Totals == nil {
		cart.Totals = zeroTotals()
	}
	if cart.Items == nil {
		cart.Items = []*api.CartItem{}
	}

	var totalItems uint32
	subtotal := big.NewRat(0, 1)
	discount := big.NewRat(0, 1)

	for _, item := range cart.Items {
		if item == nil {
			continue
		}

		if item.GetUnitPrice() == "" {
			item.UnitPrice = "0"
		}
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

func zeroTotals() *api.CartTotals {
	return &api.CartTotals{
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

func (s *service) ensureCartMutable(ctx context.Context, userID int64) error {
	active, err := s.repository.HasActiveCheckout(ctx, userID)
	if err != nil {
		return err
	}
	if active {
		return status.Error(codes.FailedPrecondition, "checkout in progress")
	}

	return nil
}
