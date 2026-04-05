package domain

import (
	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/shopspring/decimal"
)

type CartTotals struct {
	TotalItems uint32
	Subtotal   decimal.Decimal
	Discount   decimal.Decimal
	Total      decimal.Decimal
}

func ZeroTotals() *CartTotals {
	return &CartTotals{
		TotalItems: 0,
		Subtotal:   decimal.Zero,
		Discount:   decimal.Zero,
		Total:      decimal.Zero,
	}
}

func (c *CartTotals) ToProto() *client.CartTotals {
	return &client.CartTotals{
		TotalItems: c.TotalItems,
		Subtotal:   c.Subtotal.String(),
		Discount:   c.Discount.String(),
		Total:      c.Total.String(),
	}
}
