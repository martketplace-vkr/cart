package client

import rd "github.com/martketplace-vkr/cart/internal/repository/redis/client"

type Service struct {
	repository *rd.Repository
}

func New(repository *rd.Repository) *Service {
	return &Service{
		repository: repository,
	}
}
