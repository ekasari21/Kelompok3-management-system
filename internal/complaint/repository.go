package complaint

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(c *Complaint) error {
	return r.db.Create(c).Error
}

func (r *Repository) FindAll() ([]Complaint, error) {
	var complaints []Complaint

	err := r.db.
		Preload("User").
		Preload("Asset").
		Preload("Asset.Location").
		Order("created_at DESC").
		Find(&complaints).Error

	if err != nil {
		return nil, err
	}

	return complaints, nil
}

func (r *Repository) FindByID(id uuid.UUID) (*Complaint, error) {
	var complaint Complaint

	err := r.db.First(&complaint, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &complaint, nil
}

func (r *Repository) Update(c *Complaint) error {
	return r.db.Save(c).Error
}

func (r *Repository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Complaint{}, "id = ?", id).Error
}

func (r *Repository) CountByYear(year int) (int64, error) {
	var count int64
	prefix := fmt.Sprintf("CMP-%d-%%", year)

	err := r.db.Model(&Complaint{}).
		Where("code LIKE ?", prefix).
		Count(&count).Error

	return count, err
}

func (r *Repository) FindByUserID(userID uint) ([]Complaint, error) {
	var complaints []Complaint

	err := r.db.
		Preload("User").
		Preload("Asset").
		Preload("Asset.Location").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&complaints).Error

	if err != nil {
		return nil, err
	}

	return complaints, nil
}

func (r *Repository) GenerateUniqueCode(year int) (string, error) {
	var lastCode string

	prefix := fmt.Sprintf("CMP-%d-", year)

	err := r.db.
		Model(&Complaint{}).
		Where("code LIKE ?", prefix+"%").
		Order("code DESC").
		Limit(1).
		Pluck("code", &lastCode).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	nextNumber := 1

	if lastCode != "" {
		var lastNumber int

		_, err := fmt.Sscanf(lastCode, "CMP-%d-%d", new(int), &lastNumber)
		if err != nil {
			return "", fmt.Errorf("gagal membaca kode complaint terakhir %q: %w", lastCode, err)
		}

		nextNumber = lastNumber + 1
	}

	return fmt.Sprintf("CMP-%d-%03d", year, nextNumber), nil
}

func (r *Repository) MarkAsUnrepairable(id uuid.UUID, note string) (*Complaint, error) {
	var c Complaint
	err := r.db.Preload("Asset").First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Kurangi stok aset di tabel assets jika qty_in > 0
		if c.AssetID > 0 {
			_ = tx.Table("assets").
				Where("asset_id = ? AND qty_in > 0", c.AssetID).
				Update("qty_in", gorm.Expr("qty_in - 1")).Error

			// Update condition aset menjadi rusak_berat
			_ = tx.Table("assets").
				Where("asset_id = ?", c.AssetID).
				Update("condition", "rusak_berat").Error
		}

		// 2. Buat record di tabel asset_rusak (status_rusak: true = tidak bisa diperbaiki)
		now := time.Now()
		codeRusak := fmt.Sprintf("BRK-%d", now.UnixNano()/1e6)
		if len(codeRusak) > 30 {
			codeRusak = codeRusak[:30]
		}

		desc := fmt.Sprintf("Aset rusak tidak bisa diperbaiki dari keluhan %s: %s", c.Code, c.Title)
		if strings.TrimSpace(note) != "" {
			desc += ". Catatan Teknisi: " + strings.TrimSpace(note)
		}

		brokenRecord := map[string]interface{}{
			"code_rusak":   codeRusak,
			"asset_id":     c.AssetID,
			"qty_rusak":    1,
			"status":       "rusak_berat",
			"status_rusak": true, // true = tidak bisa diperbaiki / afkir
			"is_repaired":  false,
			"description":  desc,
			"create_at":    now,
			"update_at":    now,
		}

		if err := tx.Table("asset_rusak").Create(&brokenRecord).Error; err != nil {
			return err
		}

		// 3. Update status keluhan menjadi unrepairable
		c.Status = "unrepairable"
		c.IsFixed = false
		catatan := "[Teknisi: Rusak Tidak Bisa Diperbaiki]"
		if strings.TrimSpace(note) != "" {
			catatan += " " + strings.TrimSpace(note)
		}
		if c.Note != "" {
			c.Note = c.Note + "\n" + catatan
		} else {
			c.Note = catatan
		}

		return tx.Save(&c).Error
	})

	if err != nil {
		return nil, err
	}

	return &c, nil
}
