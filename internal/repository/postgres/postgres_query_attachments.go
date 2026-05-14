package postgres

import (
	"context"
	"fmt"

	m "github.com/stanterprise/observer/internal/models"
)

func (r *PostgresRepository) FindAttachmentByStorageKey(ctx context.Context, storageKey string) (map[string]interface{}, error) {
	if storageKey == "" {
		return nil, fmt.Errorf("storage key is required")
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var attempts []m.TestAttempt
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("list attempts for attachment lookup: %w", err)
	}

	for _, attempt := range attempts {
		if attachment := findAttachmentInMaps(attempt.Attachments, storageKey); attachment != nil {
			return attachment, nil
		}
		for _, failure := range attempt.Failures {
			if attachment := findAttachmentInMaps(failure.Attachments, storageKey); attachment != nil {
				return attachment, nil
			}
		}
		for _, errDoc := range attempt.Errors {
			if attachment := findAttachmentInMaps(errDoc.Attachments, storageKey); attachment != nil {
				return attachment, nil
			}
		}
	}

	return nil, nil
}
