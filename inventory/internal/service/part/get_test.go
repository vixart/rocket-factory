package part

import (
	"errors"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

func (s *ServiceSuite) TestGetSuccess() {
	var (
		partUUID = uuid.New()

		expectedPart = model.Part{
			UUID:          partUUID,
			Name:          gofakeit.Word(),
			Description:   gofakeit.Phrase(),
			Price:         int64(gofakeit.IntRange(100_00, 1_000_000_00)),
			PartType:      model.PartTypeEngine,
			StockQuantity: int64(gofakeit.IntRange(1, 100)),
			CreatedAt:     new(gofakeit.Date()),
		}
	)

	s.partRepo.
		EXPECT().
		Get(s.ctx, partUUID).
		Return(expectedPart, nil)

	res, err := s.service.Get(s.ctx, partUUID)

	s.Require().NoError(err)
	s.Require().Equal(expectedPart, res)
}

func (s *ServiceSuite) TestGetRepositoryError() {
	var (
		partUUID = uuid.New()

		repoErr = errors.New(gofakeit.Phrase())
	)

	s.partRepo.
		EXPECT().
		Get(s.ctx, partUUID).
		Return(model.Part{}, repoErr)

	res, err := s.service.Get(s.ctx, partUUID)

	s.Require().Error(err)
	s.Require().Equal(model.Part{}, res)

	s.Require().ErrorContains(
		err,
		"не удалось получить деталь",
	)

	s.Require().ErrorIs(err, repoErr)
}
