package config

import (
	catalogpb "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1"
	"github.com/martketplace-vkr/pkg/build/components/rediscomponent"
	"github.com/martketplace-vkr/pkg/server/grpc"
)

type Config struct {
	Grpc          grpc.Config           `validate:"required"`
	Redis         rediscomponent.Config `validate:"required"`
	CatalogClient catalogpb.Config      `validate:"required"`
}
