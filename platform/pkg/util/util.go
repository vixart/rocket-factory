package util

import "github.com/google/uuid"

func UUIDPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}

	return id.String()
}
