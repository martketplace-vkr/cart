package catalog

import (
	"context"
	"errors"

	"github.com/martketplace-vkr/cart/domain"
	v1 "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1"
	"github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/cart"
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

	for idx, product := range resp.Products {
		enrichErr := items[idx].EnrichByProduct(product)
		if err != nil {
			err = errors.Join(enrichErr)
		}
	}

	return err
}
