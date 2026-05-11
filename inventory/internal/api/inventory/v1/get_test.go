package v1

import (
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (s *APISuite) TestGetPartSuccess() {
	var (
		partUUID = uuid.New()

		req = &inventoryv1.GetPartRequest{
			Uuid: partUUID.String(),
		}

		part = model.Part{
			UUID:          partUUID,
			Name:          "Engine X",
			Description:   "desc",
			Price:         5000,
			PartType:      model.PartTypeEngine,
			StockQuantity: 3,
		}
	)

	s.inventoryService.
		EXPECT().
		Get(s.ctx, partUUID).
		Return(part, nil)

	res, err := s.api.GetPart(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Equal(part.UUID.String(), res.GetPart().GetUuid())
}

func (s *APISuite) TestGetPartEmptyUUID() {
	req := &inventoryv1.GetPartRequest{
		Uuid: "",
	}

	res, err := s.api.GetPart(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.InvalidArgument, st.Code())
}

func (s *APISuite) TestGetPartInvalidUUID() {
	req := &inventoryv1.GetPartRequest{
		Uuid: "not-a-uuid",
	}

	res, err := s.api.GetPart(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.InvalidArgument, st.Code())
}

func (s *APISuite) TestGetPartNotFound() {
	var (
		partUUID = uuid.New()

		req = &inventoryv1.GetPartRequest{
			Uuid: partUUID.String(),
		}
	)

	s.inventoryService.
		EXPECT().
		Get(s.ctx, partUUID).
		Return(model.Part{}, errs.ErrPartNotFound)

	res, err := s.api.GetPart(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.NotFound, st.Code())
}

func (s *APISuite) TestGetPartInternalError() {
	var (
		partUUID = uuid.New()

		req = &inventoryv1.GetPartRequest{
			Uuid: partUUID.String(),
		}

		repoErr = errors.New("db error")
	)

	s.inventoryService.
		EXPECT().
		Get(s.ctx, partUUID).
		Return(model.Part{}, repoErr)

	res, err := s.api.GetPart(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.Internal, st.Code())
}
