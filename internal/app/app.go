package app

import (
	"context"

	"github.com/martketplace-vkr/cart/config"
	"github.com/martketplace-vkr/cart/internal/app/cmp/server"
	catalogClient "github.com/martketplace-vkr/cart/internal/misc/clients/catalog"
	clientRepository "github.com/martketplace-vkr/cart/internal/repository/redis/client"
	clientService "github.com/martketplace-vkr/cart/internal/service/client"
	orderService "github.com/martketplace-vkr/cart/internal/service/order"
	clientTransport "github.com/martketplace-vkr/cart/internal/transport/grpc/v1/client"
	orderTransport "github.com/martketplace-vkr/cart/internal/transport/grpc/v1/order"
	catalogpb "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1"

	"github.com/martketplace-vkr/pkg/build"
	"github.com/martketplace-vkr/pkg/build/components/rediscomponent"
)

func Run(ctx context.Context, cfg *config.Config) error {
	redis := rediscomponent.New(cfg.Redis)
	catalogConnector := catalogpb.New(cfg.CatalogClient)

	clientRepo := clientRepository.New(redis.Client)
	clientServ := clientService.New(clientRepo, catalogClient.New(catalogConnector))
	orderServ := orderService.New(clientRepo)
	clientHandler := clientTransport.New(clientServ)
	orderHandler := orderTransport.New(orderServ)

	grpcServer := server.New(
		cfg.Grpc,
		clientHandler,
		orderHandler,
	)

	cmps := build.Components{
		redis,
		catalogConnector,
		grpcServer,
	}

	app, err := build.NewApp(cmps)
	if err != nil {
		return err
	}

	return build.Run(ctx, app)
}
