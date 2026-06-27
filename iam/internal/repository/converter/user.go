package converter

import (
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/record"
)

func UserRecordToModel(rec record.User) model.User {
	return model.User{
		UUID:         rec.UUID,
		Login:        rec.Login,
		PasswordHash: rec.PasswordHash,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
}

func UserModelToRecord(model model.User) record.User {
	return record.User{
		UUID:         model.UUID,
		Login:        model.Login,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
