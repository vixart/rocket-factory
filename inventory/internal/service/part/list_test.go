package part

import (
	"errors"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

func (s *ServiceSuite) TestListSuccess() {
	var (
		partUUID1 = uuid.New()
		partUUID2 = uuid.New()

		uuids = []uuid.UUID{
			partUUID1,
			partUUID2,
		}

		partType = model.PartTypeHull

		expectedParts = []model.Part{
			{
				UUID:          partUUID1,
				Name:          gofakeit.Word(),
				Description:   gofakeit.Phrase(),
				Price:         int64(gofakeit.IntRange(100_00, 1_000_000_00)),
				PartType:      partType,
				StockQuantity: int64(gofakeit.IntRange(1, 100)),
				CreatedAt:     new(gofakeit.Date()),
			},
			{
				UUID:          partUUID2,
				Name:          gofakeit.Word(),
				Description:   gofakeit.Phrase(),
				Price:         int64(gofakeit.IntRange(100_00, 1_000_000_00)),
				PartType:      partType,
				StockQuantity: int64(gofakeit.IntRange(1, 100)),
				CreatedAt:     new(gofakeit.Date()),
			},
		}
	)

	s.partRepo.
		EXPECT().
		List(
			s.ctx,
			model.PartFilter{
				Uuids:    uuids,
				PartType: partType,
			},
		).
		Return(expectedParts, nil)

	res, err := s.service.List(s.ctx, uuids, partType)

	s.Require().NoError(err)
	s.Require().Equal(expectedParts, res)
}

func (s *ServiceSuite) TestListRepositoryError() {
	var (
		uuids = []uuid.UUID{
			uuid.New(),
		}

		partType = model.PartTypeEngine

		repoErr = errors.New(gofakeit.Phrase())
	)

	s.partRepo.
		EXPECT().
		List(
			s.ctx,
			model.PartFilter{
				Uuids:    uuids,
				PartType: partType,
			},
		).
		Return(nil, repoErr)

	res, err := s.service.List(s.ctx, uuids, partType)

	s.Require().Error(err)
	s.Require().Empty(res)

	s.Require().ErrorContains(
		err,
		"не удалось получить детали",
	)

	s.Require().ErrorIs(err, repoErr)
}
