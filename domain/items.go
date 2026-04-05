package domain

import (
	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/shopspring/decimal"
)

type CartItem struct {
	ProductId         int64
	VendorId          int64
	VendorName        string
	ProductName       string
	ImageUrl          string
	Quantity          uint32
	AvailableQuantity uint32
	Available         bool
	Selected          bool
	UnitPrice         decimal.Decimal
	TotalPrice        decimal.Decimal
}

type CartItemList []CartItem

func (i *CartItem) ToProto() *client.CartItem {
	return &client.CartItem{
		ProductId:         i.ProductId,
		VendorId:          i.VendorId,
		VendorName:        i.VendorName,
		ProductName:       i.ProductName,
		ImageUrl:          i.ImageUrl,
		Quantity:          i.Quantity,
		AvailableQuantity: i.AvailableQuantity,
		Available:         i.Available,
		Selected:          i.Selected,
		UnitPrice:         i.UnitPrice.String(),
		TotalPrice:        i.TotalPrice.String(),
	}
}

func (l *CartItemList) ToProto() []*client.CartItem {
	protoList := make([]*client.CartItem, 0, len(*l))

	for _, domainItem := range l.ToProto() {
		protoList = append(protoList, domainItem)
	}

	return protoList
}
