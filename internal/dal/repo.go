package dal

import (
	"context"
)

type (
	Repository interface {
		Transact(ctx context.Context, txFunc func(r Repository) error) error
		WordTranslationsRepository
		CallbacksRepository
		AuthConfirmationRepository
		StatsRepository
	}
)
