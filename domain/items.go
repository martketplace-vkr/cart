package domain

import (
	"slices"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	catalogdomain "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/domain"

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

func (i *CartItem) EnrichByProduct(product *catalogdomain.Product) (err error) {
	index := slices.IndexFunc(product.Images, func(image *catalogdomain.ProductImage) bool {
		return image.IsMain
	})

	if len(product.Images) > 0 {
		i.ImageUrl = product.Images[index].GetUrl()
	}

	i.ProductName = product.Name
	i.AvailableQuantity = product.GetStockCount()
	i.UnitPrice, err = decimal.NewFromString(product.Price)
	if err != nil {
		return err
	}

	i.TotalPrice = i.UnitPrice.Mul(decimal.NewFromInt(int64(i.Quantity)))

	return nil
}

type CartItemList []CartItem

func (l *CartItemList) ToProto() []*client.CartItem {
	protoList := make([]*client.CartItem, 0, len(*l))

	for _, domainItem := range l.ToProto() {
		protoList = append(protoList, domainItem)
	}

	return protoList
}

func (l *CartItemList) ToProductIDList() []int64 {
	ids := make([]int64, 0, len(*l))

	for _, item := range *l {
		ids = append(ids, item.ProductId)
	}

	return ids
}
