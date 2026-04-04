package client

import (
	"context"
	"testing"

	api "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type repositoryStub struct {
	carts           map[int64]*api.Cart
	activeCheckouts map[int64]bool
}

func newRepositoryStub() *repositoryStub {
	return &repositoryStub{
		carts:           make(map[int64]*api.Cart),
		activeCheckouts: make(map[int64]bool),
	}
}

func (r *repositoryStub) GetCart(_ context.Context, userID int64) (*api.Cart, error) {
	cart, ok := r.carts[userID]
	if !ok {
		return nil, nil
	}

	return proto.Clone(cart).(*api.Cart), nil
}

func (r *repositoryStub) HasActiveCheckout(_ context.Context, userID int64) (bool, error) {
	return r.activeCheckouts[userID], nil
}

func (r *repositoryStub) SaveCart(_ context.Context, cart *api.Cart) error {
	r.carts[cart.GetUserId()] = proto.Clone(cart).(*api.Cart)

	return nil
}

func (r *repositoryStub) DeleteCart(_ context.Context, userID int64) error {
	delete(r.carts, userID)

	return nil
}

func TestAddItemCreatesCartAndUpdatesTotals(t *testing.T) {
	repo := newRepositoryStub()
	svc := New(repo)

	resp, err := svc.AddItem(context.Background(), &api.AddItemRequest{
		UserId:    42,
		ProductId: 1001,
		VendorId:  9,
		Quantity:  3,
	})
	if err != nil {
		t.Fatalf("AddItem returned error: %v", err)
	}

	if resp.GetCart().GetUserId() != 42 {
		t.Fatalf("unexpected user_id: %d", resp.GetCart().GetUserId())
	}
	if len(resp.GetCart().GetItems()) != 1 {
		t.Fatalf("unexpected items count: %d", len(resp.GetCart().GetItems()))
	}
	if resp.GetCart().GetTotals().GetTotalItems() != 3 {
		t.Fatalf("unexpected total_items: %d", resp.GetCart().GetTotals().GetTotalItems())
	}
	if resp.GetCart().GetTotals().GetSubtotal() != "0" {
		t.Fatalf("unexpected subtotal: %s", resp.GetCart().GetTotals().GetSubtotal())
	}
	if resp.GetCart().GetCreatedAt() == nil || resp.GetCart().GetUpdatedAt() == nil {
		t.Fatal("expected timestamps to be set")
	}
}

func TestUpdateItemQuantityRecalculatesTotalPrice(t *testing.T) {
	repo := newRepositoryStub()
	repo.carts[7] = &api.Cart{
		Id:        7,
		UserId:    7,
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
		Items: []*api.CartItem{
			{
				ProductId:         501,
				VendorId:          3,
				Quantity:          1,
				AvailableQuantity: 1,
				Available:         true,
				Selected:          true,
				UnitPrice:         "12.5",
				TotalPrice:        "12.5",
			},
		},
		Totals: &api.CartTotals{
			TotalItems: 1,
			Subtotal:   "12.5",
			Discount:   "0",
			Total:      "12.5",
		},
	}

	svc := New(repo)

	resp, err := svc.UpdateItemQuantity(context.Background(), &api.UpdateItemQuantityRequest{
		UserId:    7,
		ProductId: 501,
		Quantity:  4,
	})
	if err != nil {
		t.Fatalf("UpdateItemQuantity returned error: %v", err)
	}

	item := resp.GetCart().GetItems()[0]
	if item.GetQuantity() != 4 {
		t.Fatalf("unexpected quantity: %d", item.GetQuantity())
	}
	if item.GetTotalPrice() != "50" {
		t.Fatalf("unexpected total_price: %s", item.GetTotalPrice())
	}
	if resp.GetCart().GetTotals().GetTotal() != "50" {
		t.Fatalf("unexpected total: %s", resp.GetCart().GetTotals().GetTotal())
	}
}

func TestRemoveMissingItemReturnsNotFound(t *testing.T) {
	repo := newRepositoryStub()
	repo.carts[1] = &api.Cart{
		Id:     1,
		UserId: 1,
		Items:  []*api.CartItem{},
		Totals: zeroTotals(),
	}

	svc := New(repo)

	_, err := svc.RemoveItem(context.Background(), &api.RemoveItemRequest{
		UserId:    1,
		ProductId: 999,
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestClearCartDeletesStoredCart(t *testing.T) {
	repo := newRepositoryStub()
	repo.carts[5] = &api.Cart{
		Id:     5,
		UserId: 5,
		Items: []*api.CartItem{
			{ProductId: 11, Quantity: 2, UnitPrice: "3", TotalPrice: "6"},
		},
		Totals: &api.CartTotals{
			TotalItems: 2,
			Subtotal:   "6",
			Discount:   "0",
			Total:      "6",
		},
	}

	svc := New(repo)

	resp, err := svc.ClearCart(context.Background(), &api.ClearCartRequest{UserId: 5})
	if err != nil {
		t.Fatalf("ClearCart returned error: %v", err)
	}

	if len(resp.GetCart().GetItems()) != 0 {
		t.Fatalf("expected empty cart, got %d items", len(resp.GetCart().GetItems()))
	}
	if _, ok := repo.carts[5]; ok {
		t.Fatal("expected cart to be removed from repository")
	}
}

func TestAddItemBlockedWhileCheckoutInProgress(t *testing.T) {
	repo := newRepositoryStub()
	repo.activeCheckouts[42] = true

	svc := New(repo)

	_, err := svc.AddItem(context.Background(), &api.AddItemRequest{
		UserId:    42,
		ProductId: 1001,
		VendorId:  9,
		Quantity:  1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}
