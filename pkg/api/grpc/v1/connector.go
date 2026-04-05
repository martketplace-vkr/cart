package v1

import (
	"context"
	"time"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/admin"
	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	cmpName = "CatalogClientGrpc"
)

type CartClient struct {
	Admin  admin.CartAdminServiceClient
	Client client.CartClientServiceClient

	conn *grpc.ClientConn
	cfg  Config
}

func New(cfg Config) *CartClient {
	return &CartClient{cfg: cfg}
}

func (c *CartClient) Start(ctx context.Context) (err error) {
	if c.cfg.DontRun {
		return nil
	}

	options := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	if c.cfg.Retry != nil {
		options = append(options, grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  c.cfg.Retry.BaseDelay,
				Multiplier: c.cfg.Retry.Multiplier,
				Jitter:     c.cfg.Retry.Jitter,
				MaxDelay:   c.cfg.Retry.MaxDelay,
			},
		}))
	}

	c.conn, err = grpc.DialContext(
		ctx,
		c.cfg.Address,
		options...,
	)
	if err != nil {
		return err
	}

	c.Client = client.NewCartClientServiceClient(c.conn)
	c.Admin = admin.NewCartAdminServiceClient(c.conn)

	return nil
}

func (c *CartClient) Stop(_ context.Context) error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *CartClient) GetStartTimeout() time.Duration {
	return c.cfg.StartTimeout.Duration
}

func (c *CartClient) GetStopTimeout() time.Duration {
	return c.cfg.StopTimeout.Duration
}

func (c *CartClient) GetShutdownDelay() time.Duration {
	return c.cfg.ShutdownDelay.Duration
}

func (c *CartClient) GetName() string {
	return cmpName
}
