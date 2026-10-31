package assetlending

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) GetDB() *gorm.DB       { return r.db }

func (r *Repository) Create(ctx context.Context, tx *gorm.DB, item *LendingAsset) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(item).Error
}

func (r *Repository) queryWithRelations(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.WithContext(ctx).
		Preload("User").
		Preload("Asset").
		Preload("Asset.Category").
		Preload("Asset.Location")
}

func (r *Repository) List(ctx context.Context) ([]LendingAsset, error) {
	var items []LendingAsset
	err := r.queryWithRelations(ctx, r.db).Order("lending_id DESC").Find(&items).Error
	return items, err
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*LendingAsset, error) {
	var item LendingAsset
	err := r.queryWithRelations(ctx, r.db).First(&item, "lending_id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) Update(ctx context.Context, tx *gorm.DB, id uint, input *LendingAsset) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	updates := map[string]interface{}{
		"user_id":        input.UserID,
		"asset_id":       input.AssetID,
		"qty_lending":    input.QtyLending,
		"lending_date":   input.LendingDate,
		"lending_return": input.ReturnDate,
		"lending_status": input.Status,
	}
	result := db.WithContext(ctx).Model(&LendingAsset{}).Where("lending_id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ReturnAsset hanya mengubah transaksi yang belum berstatus dikembalikan.
// Jika tidak ada baris yang berubah, service harus membatalkan transaksi stok.
func (r *Repository) ReturnAsset(ctx context.Context, tx *gorm.DB, id uint, returnDate time.Time) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	result := db.WithContext(ctx).Model(&LendingAsset{}).
		Where("lending_id = ? AND lending_status <> ?", id, StatusDikembalikan).
		Updates(map[string]interface{}{
			"lending_return": returnDate,
			"lending_status": StatusDikembalikan,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := db.WithContext(ctx).Model(&LendingAsset{}).Where("lending_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		return ErrAlreadyReturned
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, tx *gorm.DB, id uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	result := db.WithContext(ctx).Where("lending_id = ?", id).Delete(&LendingAsset{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uint) ([]LendingAsset, error) {
	var items []LendingAsset
	err := r.queryWithRelations(ctx, r.db).
		Where("user_id = ?", userID).
		Order("lending_id DESC").
		Find(&items).Error
	return items, err
}
