package order

type service struct {
	repository repository
}

func New(repository repository) *service {
	return &service{
		repository: repository,
	}
}
