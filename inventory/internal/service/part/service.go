package part

type service struct {
	partRepo Repository
}

func NewService(partRepo Repository) *service {
	return &service{
		partRepo: partRepo,
	}
}
