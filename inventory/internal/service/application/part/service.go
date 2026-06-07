package part

type service struct {
	txManager            TxManager
	partRepo             Repository
	compatibilityChecker CompatibilityChecker
}

func NewService(txManager TxManager, partRepo Repository, checker CompatibilityChecker) *service {
	return &service{
		txManager:            txManager,
		partRepo:             partRepo,
		compatibilityChecker: checker,
	}
}
