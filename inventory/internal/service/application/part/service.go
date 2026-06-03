package part

type service struct {
	partRepo             Repository
	compatibilityChecker CompatibilityChecker
}

func NewService(partRepo Repository, checker CompatibilityChecker) *service {
	return &service{
		partRepo:             partRepo,
		compatibilityChecker: checker,
	}
}
