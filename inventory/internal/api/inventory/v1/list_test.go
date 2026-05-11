package v1

import (
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/converter"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (s *APISuite) TestListPartsSuccess() {
	var (
		partUUID1 = uuid.New()
		partUUID2 = uuid.New()

		req = &inventoryv1.ListPartsRequest{
			Uuids: []string{
				partUUID1.String(),
				partUUID2.String(),
			},
			PartType: inventoryv1.PartType_PART_TYPE_HULL,
		}

		expectedUuids = []uuid.UUID{partUUID1, partUUID2}
		expectedType  = converter.PartTypeProtoToModel(req.GetPartType())

		parts = []model.Part{
			{
				UUID:          partUUID1,
				Name:          "Hull A",
				Description:   "desc",
				Price:         1000,
				PartType:      expectedType,
				StockQuantity: 10,
			},
			{
				UUID:          partUUID2,
				Name:          "Hull B",
				Description:   "desc",
				Price:         2000,
				PartType:      expectedType,
				StockQuantity: 5,
			},
		}
	)

	s.inventoryService.
		EXPECT().
		List(s.ctx, expectedUuids, expectedType).
		Return(parts, nil)

	res, err := s.api.ListParts(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Len(res.Parts, 2)
}

func (s *APISuite) TestListPartsInvalidUUID() {
	req := &inventoryv1.ListPartsRequest{
		Uuids: []string{"not-a-uuid"},
	}

	res, err := s.api.ListParts(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.InvalidArgument, st.Code())
}

func (s *APISuite) TestListPartsNotFound() {
	var (
		partUUID = uuid.New()

		req = &inventoryv1.ListPartsRequest{
			Uuids: []string{partUUID.String()},
		}

		repoErr = errs.ErrPartNotFound
	)

	s.inventoryService.
		EXPECT().
		List(s.ctx, []uuid.UUID{partUUID}, converter.PartTypeProtoToModel(req.GetPartType())).
		Return(nil, repoErr)

	res, err := s.api.ListParts(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.NotFound, st.Code())
}

func (s *APISuite) TestListPartsInternalError() {
	var (
		partUUID = uuid.New()

		req = &inventoryv1.ListPartsRequest{
			Uuids: []string{partUUID.String()},
		}

		repoErr = errors.New("db is down")
	)

	s.inventoryService.
		EXPECT().
		List(s.ctx, []uuid.UUID{partUUID}, converter.PartTypeProtoToModel(req.GetPartType())).
		Return(nil, repoErr)

	res, err := s.api.ListParts(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.Internal, st.Code())
}
