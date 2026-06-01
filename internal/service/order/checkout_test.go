package order

import (
	"context"
	"testing"

	orderapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type repositoryStub struct {
	reservation *orderapi.CheckoutReservation
	commitResp  *orderapi.CommitCheckoutResponse
	releaseResp *orderapi.ReleaseCheckoutResponse
}

func (r *repositoryStub) ReserveCheckoutItems(
	_ context.Context,
	_ int64,
	_ string,
	_ []int64,
	_ uint64,
	_ int64,
) (*orderapi.CheckoutReservation, error) {
	return r.reservation, nil
}

func (r *repositoryStub) GetCheckoutReservation(
	_ context.Context,
	_ int64,
	_ string,
) (*orderapi.CheckoutReservation, error) {
	return r.reservation, nil
}

func (r *repositoryStub) CommitCheckout(_ context.Context, _ int64, _ string) (*orderapi.CommitCheckoutResponse, error) {
	return r.commitResp, nil
}

func (r *repositoryStub) ReleaseCheckout(_ context.Context, _ int64, _ string, _ string) (*orderapi.ReleaseCheckoutResponse, error) {
	return r.releaseResp, nil
}

func TestReserveCheckoutItemsReturnsReservation(t *testing.T) {
	repo := &repositoryStub{
		reservation: &orderapi.CheckoutReservation{
			CheckoutId: "chk-1",
			UserId:     42,
			CreatedAt:  timestamppb.Now(),
		},
	}

	svc := New(repo)

	resp, err := svc.ReserveCheckoutItems(context.Background(), &orderapi.ReserveCheckoutItemsRequest{
		UserId:              42,
		CheckoutId:          "chk-1",
		ProductIds:          []int64{1001},
		ExpectedCartVersion: 11,
	})
	if err != nil {
		t.Fatalf("ReserveCheckoutItems returned error: %v", err)
	}
	if resp.GetReservation().GetCheckoutId() != "chk-1" {
		t.Fatalf("unexpected checkout_id: %s", resp.GetReservation().GetCheckoutId())
	}
}

func TestReserveCheckoutItemsValidatesInput(t *testing.T) {
	svc := New(&repositoryStub{})

	_, err := svc.ReserveCheckoutItems(context.Background(), &orderapi.ReserveCheckoutItemsRequest{
		UserId:     42,
		CheckoutId: "chk-1",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestCommitCheckoutReturnsRepositoryResponse(t *testing.T) {
	repo := &repositoryStub{
		commitResp: &orderapi.CommitCheckoutResponse{
			UserId:            42,
			CheckoutId:        "chk-1",
			RemovedProductIds: []int64{1001},
			CommittedAt:       timestamppb.Now(),
		},
	}

	svc := New(repo)

	resp, err := svc.CommitCheckout(context.Background(), &orderapi.CommitCheckoutRequest{
		UserId:     42,
		CheckoutId: "chk-1",
	})
	if err != nil {
		t.Fatalf("CommitCheckout returned error: %v", err)
	}
	if len(resp.GetRemovedProductIds()) != 1 || resp.GetRemovedProductIds()[0] != 1001 {
		t.Fatalf("unexpected removed_product_ids: %v", resp.GetRemovedProductIds())
	}
}
