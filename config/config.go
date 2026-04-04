package config

import (
	"github.com/martketplace-vkr/pkg/build/components/rediscomponent"
	"github.com/martketplace-vkr/pkg/server/grpc"
)

type Config struct {
	Grpc  grpc.Config           `validate:"required"`
	Redis rediscomponent.Config `validate:"required"`
}
