package domain

import (
	"time"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/shopspring/decimal"
)

type Cart struct {
	Id        int64
	UserId    int64
	Items     CartItemList
	Totals    *CartTotals
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func EmptyCart(userID int64) *Cart {
	return &Cart{
		Id:     userID,
		UserId: userID,
		Items:  []CartItem{},
		Totals: ZeroTotals(),
	}
}

func (c *Cart) ToProto() *client.Cart {
	return &client.Cart{
		Id:     c.Id,
		UserId: c.UserId,
		Items:  c.Items.ToProto(),
		Totals: c.Totals.ToProto(),
	}
}

func (c *Cart) Recalculate() {
	discount := decimal.Zero
	if c.Totals != nil {
		discount = c.Totals.Discount
	}
	c.Totals = ZeroTotals()
	c.Totals.Discount = discount

	if c.Items == nil {
		c.Items = CartItemList{}
	}

	for idx, item := range c.Items {
		item.TotalPrice = item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		c.Totals.TotalItems += item.Quantity

		if item.Selected {
			c.Totals.Subtotal = c.Totals.Subtotal.Add(item.TotalPrice)
		}

		c.Items[idx] = item
	}

	c.Totals.Total = c.Totals.Subtotal.Sub(c.Totals.Discount)
}

func (c *Cart) PrepareForSave() {
	now := time.Now()

	if c.Id == 0 {
		c.Id = c.UserId
	}

	if c.CreatedAt == nil {
		c.CreatedAt = &now
	}

	c.Recalculate()
	c.UpdatedAt = &now
}
