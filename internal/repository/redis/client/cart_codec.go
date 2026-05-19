package client

import (
	"encoding/json"
	"time"

	"github.com/martketplace-vkr/cart/domain"
	clientapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func unmarshalDomainCartPayload(payload string) (*domain.Cart, error) {
	cart := &domain.Cart{}
	if err := json.Unmarshal([]byte(payload), cart); err == nil {
		return normalizeDomainCart(cart), nil
	}

	protoCart := &clientapi.Cart{}
	if err := protojson.Unmarshal([]byte(payload), protoCart); err != nil {
		return nil, err
	}

	return protoCartToDomain(protoCart)
}

func unmarshalProtoCartPayload(payload string) (*clientapi.Cart, error) {
	cart, err := unmarshalDomainCartPayload(payload)
	if err != nil {
		return nil, err
	}

	return cart.ToProto(), nil
}

func marshalProtoCartPayload(cart *clientapi.Cart) ([]byte, error) {
	domainCart, err := protoCartToDomain(cart)
	if err != nil {
		return nil, err
	}

	return json.Marshal(domainCart)
}

func protoCartToDomain(cart *clientapi.Cart) (*domain.Cart, error) {
	if cart == nil {
		return normalizeDomainCart(&domain.Cart{}), nil
	}

	domainCart := &domain.Cart{
		Id:        cart.GetId(),
		UserId:    cart.GetUserId(),
		Items:     make(domain.CartItemList, 0, len(cart.GetItems())),
		Totals:    domain.ZeroTotals(),
		CreatedAt: protoTime(cart.GetCreatedAt()),
		UpdatedAt: protoTime(cart.GetUpdatedAt()),
	}

	if cart.GetTotals() != nil {
		domainCart.Totals = &domain.CartTotals{
			TotalItems: cart.GetTotals().GetTotalItems(),
			Subtotal:   decimalFromString(cart.GetTotals().GetSubtotal()),
			Discount:   decimalFromString(cart.GetTotals().GetDiscount()),
			Total:      decimalFromString(cart.GetTotals().GetTotal()),
		}
	}

	for _, item := range cart.GetItems() {
		if item == nil {
			continue
		}

		domainCart.Items = append(domainCart.Items, domain.CartItem{
			ProductId:         item.GetProductId(),
			VendorId:          item.GetVendorId(),
			VendorName:        item.GetVendorName(),
			ProductName:       item.GetProductName(),
			ImageUrl:          item.GetImageUrl(),
			Quantity:          item.GetQuantity(),
			AvailableQuantity: item.GetAvailableQuantity(),
			Available:         item.GetAvailable(),
			Selected:          item.GetSelected(),
			UnitPrice:         decimalFromString(item.GetUnitPrice()),
			TotalPrice:        decimalFromString(item.GetTotalPrice()),
		})
	}

	return normalizeDomainCart(domainCart), nil
}

func normalizeDomainCart(cart *domain.Cart) *domain.Cart {
	if cart.Totals == nil {
		cart.Totals = domain.ZeroTotals()
	}
	if cart.Items == nil {
		cart.Items = domain.CartItemList{}
	}

	return cart
}

func decimalFromString(value string) decimal.Decimal {
	parsed, err := decimal.NewFromString(normalizeDecimalString(value))
	if err != nil {
		return decimal.Zero
	}

	return parsed
}

func protoTime(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}

	result := value.AsTime()
	return &result
}
