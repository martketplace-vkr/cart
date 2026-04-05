package client

import (
	"github.com/martketplace-vkr/cart/internal/misc/clients/catalog"
	rd "github.com/martketplace-vkr/cart/internal/repository/redis/client"
)

type Service struct {
	repository *rd.Repository
	catalog    *catalog.Client
}

func New(repository *rd.Repository) *Service {
	return &Service{
		repository: repository,
	}
}
