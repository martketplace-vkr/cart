package client

import (
	"testing"

	clientapi "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/pkg/utils/currency"
)

func TestCheckoutCurrencyAndUnitPrice(t *testing.T) {
	tests := []struct {
		name                string
		item                *clientapi.CartItem
		preferredCurrencyID int64
		wantCurrencyID      int64
		wantPrice           string
	}{
		{
			name:                "crypto product uses USDT",
			item:                &clientapi.CartItem{RubPrice: "9000", UsdtPrice: "100", AcceptsCrypto: true},
			preferredCurrencyID: int64(currency.USDTinTRC),
			wantCurrencyID:      int64(currency.USDTinTRC),
			wantPrice:           "100",
		},
		{
			name:                "regular product falls back to RUB",
			item:                &clientapi.CartItem{RubPrice: "9000"},
			preferredCurrencyID: int64(currency.USDTinTRC),
			wantCurrencyID:      int64(currency.RUB),
			wantPrice:           "9000",
		},
		{
			name:                "RUB preference keeps RUB",
			item:                &clientapi.CartItem{RubPrice: "9000", UsdtPrice: "100", AcceptsCrypto: true},
			preferredCurrencyID: int64(currency.RUB),
			wantCurrencyID:      int64(currency.RUB),
			wantPrice:           "9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currencyID, price := checkoutCurrencyAndUnitPrice(tt.item, tt.preferredCurrencyID)
			if currencyID != tt.wantCurrencyID || price != tt.wantPrice {
				t.Fatalf("checkoutCurrencyAndUnitPrice() = (%d, %q), want (%d, %q)", currencyID, price, tt.wantCurrencyID, tt.wantPrice)
			}
		})
	}
}
