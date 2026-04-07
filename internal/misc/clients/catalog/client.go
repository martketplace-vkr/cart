package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/martketplace-vkr/cart/domain"
	v1 "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1"
	"github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/cart"
	catalogdomain "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/domain"
)

type Client struct {
	client *v1.Client
}

func New(client *v1.Client) *Client {
	return &Client{
		client: client,
	}
}

func (c *Client) EnrichCartItemByProductInfo(ctx context.Context, items domain.CartItemList) (err error) {
	resp, err := c.client.Cart.GetProductList(ctx, &cart.GetProductListRequest{
		ProductIds: items.ToProductIDList(),
	})
	if err != nil {
		return err
	}

	productsByID := make(map[int64]*catalogdomain.Product, len(resp.GetProducts()))
	for _, product := range resp.GetProducts() {
		if product != nil {
			productsByID[product.GetId()] = product
		}
	}

	for idx := range items {
		product, ok := productsByID[items[idx].ProductId]
		if !ok {
			err = errors.Join(err, fmt.Errorf("product %d not found in catalog response", items[idx].ProductId))
			continue
		}

		if enrichErr := items[idx].EnrichByProduct(product); enrichErr != nil {
			err = errors.Join(err, fmt.Errorf("enrich product %d: %w", items[idx].ProductId, enrichErr))
		}
	}

	return err
}
