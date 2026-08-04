package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestPart_ReserveAndRelease(t *testing.T) {
	t.Parallel()

	partUUID := uuid.New()
	now := time.Now()

	hullProps, err := valueobject.NewHullProperties(100)
	require.NoError(t, err)

	part := entity.RestorePart(
		partUUID,
		"test part",
		"desc",
		valueobject.PartTypeHull,
		1000,
		5,
		0,
		hullProps,
		now,
	)

	// --- Reserve success ---
	err = part.Reserve()
	require.NoError(t, err)

	assert.Equal(t, 1, part.Reserved())
	assert.Equal(t, 5, part.StockQuantity()) // stock stays the same

	// --- Release success ---
	err = part.Release()
	require.NoError(t, err)

	assert.Equal(t, 0, part.Reserved())
}

func TestPart_Reserve_OutOfStock(t *testing.T) {
	t.Parallel()

	partUUID := uuid.New()
	now := time.Now()

	hullProps, err := valueobject.NewHullProperties(100)
	require.NoError(t, err)

	part := entity.RestorePart(
		partUUID,
		"test part",
		"desc",
		valueobject.PartTypeHull,
		1000,
		1,
		1, // already reserved => no free stock
		hullProps,
		now,
	)

	err = part.Reserve()

	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrOutOfStock)
}

func TestPart_Release_Empty(t *testing.T) {
	t.Parallel()

	partUUID := uuid.New()
	now := time.Now()

	hullProps, err := valueobject.NewHullProperties(100)
	require.NoError(t, err)

	part := entity.RestorePart(
		partUUID,
		"test part",
		"desc",
		valueobject.PartTypeHull,
		1000,
		5,
		0,
		hullProps,
		now,
	)

	err = part.Release()

	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrNothingToRelease)
}
