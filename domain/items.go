package domain

import (
	"slices"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	catalogdomain "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/domain"
	"github.com/martketplace-vkr/pkg/utils/currency"

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
	CurrencyID        int64
	UnitPrice         decimal.Decimal
	TotalPrice        decimal.Decimal
	RubPrice          decimal.Decimal
	USDTPrice         decimal.Decimal
	RubPerUSDT        decimal.Decimal
	AcceptsCrypto     bool
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
		CurrencyId:        i.CurrencyID,
		UnitPrice:         i.UnitPrice.String(),
		TotalPrice:        i.TotalPrice.String(),
		RubPrice:          i.RubPrice.String(),
		UsdtPrice:         i.USDTPrice.String(),
		RubPerUsdt:        i.RubPerUSDT.String(),
		AcceptsCrypto:     i.AcceptsCrypto,
	}
}

func (i *CartItem) EnrichByProduct(product *catalogdomain.Product) (err error) {
	images := product.GetImages()
	if len(images) > 0 {
		index := slices.IndexFunc(images, func(image *catalogdomain.ProductImage) bool {
			return image.GetIsMain()
		})
		if index < 0 {
			index = 0
		}
		i.ImageUrl = images[index].GetUrl()
	}

	i.ProductName = product.GetName()
	i.AvailableQuantity = product.GetStockCount()
	i.RubPrice, err = decimal.NewFromString(product.GetPrice())
	if err != nil {
		return err
	}

	i.AcceptsCrypto = product.GetAcceptsCrypto()
	i.USDTPrice = decimal.Zero
	if product.GetEffectiveUsdtPrice() != "" {
		i.USDTPrice, _ = decimal.NewFromString(product.GetEffectiveUsdtPrice())
	}
	i.RubPerUSDT = decimal.Zero
	if product.GetRubPerUsdt() != "" {
		i.RubPerUSDT, _ = decimal.NewFromString(product.GetRubPerUsdt())
	}

	i.CurrencyID = int64(currency.RUB)
	i.UnitPrice = i.RubPrice
	i.TotalPrice = i.UnitPrice.Mul(decimal.NewFromInt(int64(i.Quantity)))

	return nil
}

type CartItemList []CartItem

func (l *CartItemList) ToProto() []*client.CartItem {
	protoList := make([]*client.CartItem, 0, len(*l))

	for _, domainItem := range *l {
		protoList = append(protoList, domainItem.ToProto())
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
