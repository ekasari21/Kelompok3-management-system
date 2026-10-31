package chatbot

import (
	"context"
	"strings"

	"gin-auth/internal/asset"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetInventorySummary mengambil ringkasan total aset, ketersediaan, peminjaman, dan kerusakan.
func (r *Repository) GetInventorySummary(ctx context.Context) (*InventorySummary, error) {
	summary := &InventorySummary{}
	if r.db == nil {
		return summary, nil
	}

	db := r.db.WithContext(ctx)

	// 1. Total Jenis Aset Aktif
	_ = db.Table("assets").
		Where("status = ? OR status IS TRUE", true).
		Count(&summary.TotalJenisAset).Error

	// 2. Total Keseluruhan Unit Aset Aktif
	_ = db.Table("assets").
		Where("status = ? OR status IS TRUE", true).
		Select("COALESCE(SUM(qty_in), 0)").
		Scan(&summary.TotalUnitAset).Error

	// Fallback jika belum ada data yang status-nya true secara eksplisit
	if summary.TotalJenisAset == 0 {
		_ = db.Table("assets").Count(&summary.TotalJenisAset).Error
		_ = db.Table("assets").Select("COALESCE(SUM(qty_in), 0)").Scan(&summary.TotalUnitAset).Error
	}

	// 3. Total Unit Sedang Dipinjam
	_ = db.Table("lending_assets").
		Where("LOWER(lending_status) IN ?", []string{"dipinjam", "terlambat"}).
		Select("COALESCE(SUM(qty_lending), 0)").
		Scan(&summary.AsetDipinjam).Error

	// 4. Total Unit Rusak
	_ = db.Table("asset_rusak").
		Select("COALESCE(SUM(qty_rusak), 0)").
		Scan(&summary.AsetRusak).Error

	// 5. Unit Rusak yang Bisa Diperbaiki
	_ = db.Table("asset_rusak").
		Where("is_repaired = ? OR status_rusak = ?", true, false).
		Select("COALESCE(SUM(qty_rusak), 0)").
		Scan(&summary.RusakBisaPerbaiki).Error

	// 6. Unit Rusak yang Tidak Bisa Diperbaiki (Rusak Berat / Afkir)
	_ = db.Table("asset_rusak").
		Where("status_rusak = ?", true).
		Select("COALESCE(SUM(qty_rusak), 0)").
		Scan(&summary.RusakTidakBisa).Error

	// 7. Hitung Unit Tersedia (Total - Dipinjam - Rusak)
	available := summary.TotalUnitAset - (summary.AsetDipinjam + summary.AsetRusak)
	if available < 0 {
		available = 0
	}
	summary.AsetTersedia = available

	// 8. Total Kategori
	_ = db.Table("categories").Count(&summary.TotalKategori).Error

	// 9. Total Lokasi Aktif
	_ = db.Table("locations").
		Where("is_active = ?", true).
		Count(&summary.TotalLokasi).Error

	return summary, nil
}

// SearchAssets mencari aset berdasarkan nama atau kode barang beserta ketersediaan stoknya.
func (r *Repository) SearchAssets(ctx context.Context, keyword string, limit int) ([]AssetDetailInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var assets []asset.Asset
	db := r.db.WithContext(ctx)

	term := "%" + strings.ToLower(strings.TrimSpace(keyword)) + "%"
	err := db.Preload("Category").
		Preload("Location").
		Where("status = ?", true).
		Where("LOWER(nama_asset) LIKE ? OR LOWER(code_asset) LIKE ?", term, term).
		Limit(limit).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}

	if len(assets) == 0 {
		return []AssetDetailInfo{}, nil
	}

	return r.populateAssetQty(ctx, assets)
}

// GetAvailableAssets mengambil daftar aset yang memiliki stok siap pakai/pinjam.
func (r *Repository) GetAvailableAssets(ctx context.Context, limit int) ([]AssetDetailInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var assets []asset.Asset
	db := r.db.WithContext(ctx)

	err := db.Preload("Category").
		Preload("Location").
		Where("status = ?", true).
		Order("asset_id DESC").
		Limit(limit * 2). // Ambil buffer lalu filter yang tersedia
		Find(&assets).Error
	if err != nil {
		return nil, err
	}

	allDetails, err := r.populateAssetQty(ctx, assets)
	if err != nil {
		return nil, err
	}

	var availableList []AssetDetailInfo
	for _, item := range allDetails {
		if item.QtyTersedia > 0 {
			availableList = append(availableList, item)
			if len(availableList) >= limit {
				break
			}
		}
	}

	return availableList, nil
}

// GetBrokenAssets mengambil daftar unit aset yang dilaporkan rusak.
func (r *Repository) GetBrokenAssets(ctx context.Context, statusFilter string, limit int) ([]BrokenInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	type rawBroken struct {
		RusakID     uint   `gorm:"column:rusak_id"`
		CodeRusak   string `gorm:"column:code_rusak"`
		NamaAsset   string `gorm:"column:nama_asset"`
		CodeAsset   string `gorm:"column:code_asset"`
		QtyRusak    int    `gorm:"column:qty_rusak"`
		Status      string `gorm:"column:status"`
		IsRepaired  bool   `gorm:"column:is_repaired"`
		Description string `gorm:"column:description"`
	}

	var rows []rawBroken
	query := r.db.WithContext(ctx).
		Table("asset_rusak AS ar").
		Select("ar.rusak_id, ar.code_rusak, a.nama_asset, a.code_asset, ar.qty_rusak, ar.status, ar.is_repaired, ar.description").
		Joins("JOIN assets AS a ON a.asset_id = ar.asset_id").
		Order("ar.rusak_id DESC")

	if statusFilter != "" {
		query = query.Where("LOWER(ar.status) LIKE ?", "%"+strings.ToLower(statusFilter)+"%")
	}

	err := query.Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]BrokenInfo, len(rows))
	for i, row := range rows {
		results[i] = BrokenInfo{
			RusakID:     row.RusakID,
			CodeRusak:   row.CodeRusak,
			NamaAsset:   row.NamaAsset,
			CodeAsset:   row.CodeAsset,
			QtyRusak:    row.QtyRusak,
			Status:      row.Status,
			IsRepaired:  row.IsRepaired,
			Description: row.Description,
		}
	}

	return results, nil
}

// GetActiveLendings mengambil daftar peminjaman aset yang belum dikembalikan.
func (r *Repository) GetActiveLendings(ctx context.Context, limit int) ([]LendingInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	type rawLending struct {
		LendingID   uint    `gorm:"column:lending_id"`
		Borrower    string  `gorm:"column:borrower"`
		NamaAsset   string  `gorm:"column:nama_asset"`
		CodeAsset   string  `gorm:"column:code_asset"`
		Qty         int     `gorm:"column:qty_lending"`
		LendingDate string  `gorm:"column:lending_date"`
		ReturnDate  *string `gorm:"column:lending_return"`
		Status      string  `gorm:"column:lending_status"`
	}

	var rows []rawLending
	err := r.db.WithContext(ctx).
		Table("lending_assets AS la").
		Select("la.lending_id, COALESCE(u.name, 'Pengguna') AS borrower, a.nama_asset, a.code_asset, la.qty_lending, la.lending_date, la.lending_return, la.lending_status").
		Joins("JOIN assets AS a ON a.asset_id = la.asset_id").
		Joins("LEFT JOIN users AS u ON u.id = la.user_id").
		Where("LOWER(la.lending_status) IN ?", []string{"dipinjam", "terlambat"}).
		Order("la.lending_id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]LendingInfo, len(rows))
	for i, r := range rows {
		results[i] = LendingInfo{
			LendingID: r.LendingID,
			Borrower:  r.Borrower,
			NamaAsset: r.NamaAsset,
			CodeAsset: r.CodeAsset,
			Qty:       r.Qty,
			Status:    r.Status,
		}
	}

	return results, nil
}

// GetLocationsWithCount mengambil seluruh daftar lokasi dan jumlah aset di dalamnya.
func (r *Repository) GetLocationsWithCount(ctx context.Context) ([]LocationItem, error) {
	if r.db == nil {
		return nil, nil
	}

	var items []LocationItem
	err := r.db.WithContext(ctx).
		Table("locations AS l").
		Select("l.name, l.description, COUNT(a.asset_id) AS asset_count").
		Joins("LEFT JOIN assets AS a ON a.location = l.id AND a.status = true").
		Where("l.is_active = ?", true).
		Group("l.id, l.name, l.description").
		Order("asset_count DESC, l.name ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetCategoriesWithCount mengambil seluruh kategori beserta jumlah aset di dalamnya.
func (r *Repository) GetCategoriesWithCount(ctx context.Context) ([]CategoryItem, error) {
	if r.db == nil {
		return nil, nil
	}

	var items []CategoryItem
	err := r.db.WithContext(ctx).
		Table("categories AS c").
		Select("c.name, COUNT(a.asset_id) AS asset_count").
		Joins("LEFT JOIN assets AS a ON a.category = c.id AND a.status = true").
		Group("c.id, c.name").
		Order("asset_count DESC, c.name ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetAssetsByLocation mencari daftar aset pada lokasi atau ruangan tertentu.
func (r *Repository) GetAssetsByLocation(ctx context.Context, locKeyword string, limit int) ([]AssetDetailInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	term := "%" + strings.ToLower(strings.TrimSpace(locKeyword)) + "%"
	var assets []asset.Asset

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Location").
		Joins("JOIN locations AS l ON l.id = assets.location").
		Where("assets.status = ? AND LOWER(l.name) LIKE ?", true, term).
		Limit(limit).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}

	return r.populateAssetQty(ctx, assets)
}

// GetAssetsByCategory mencari daftar aset berdasarkan kategori tertentu.
func (r *Repository) GetAssetsByCategory(ctx context.Context, catKeyword string, limit int) ([]AssetDetailInfo, error) {
	if r.db == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	term := "%" + strings.ToLower(strings.TrimSpace(catKeyword)) + "%"
	var assets []asset.Asset

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Location").
		Joins("JOIN categories AS c ON c.id = assets.category").
		Where("assets.status = ? AND LOWER(c.name) LIKE ?", true, term).
		Limit(limit).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}

	return r.populateAssetQty(ctx, assets)
}

// populateAssetQty menghitung stok rusak, dipinjam, dan sisa tersedia untuk sekelompok aset.
func (r *Repository) populateAssetQty(ctx context.Context, assets []asset.Asset) ([]AssetDetailInfo, error) {
	if len(assets) == 0 {
		return []AssetDetailInfo{}, nil
	}

	assetIDs := make([]uint, len(assets))
	for i, a := range assets {
		assetIDs[i] = a.AssetID
	}

	type qtyResult struct {
		AssetID uint `gorm:"column:asset_id"`
		Total   int  `gorm:"column:total"`
	}

	// 1. Hitung jumlah yang sedang dipinjam
	var lendingResults []qtyResult
	_ = r.db.WithContext(ctx).
		Table("lending_assets").
		Select("asset_id, COALESCE(SUM(qty_lending), 0) AS total").
		Where("asset_id IN ? AND LOWER(lending_status) IN ?", assetIDs, []string{"dipinjam", "terlambat"}).
		Group("asset_id").
		Scan(&lendingResults).Error

	lendingMap := make(map[uint]int, len(lendingResults))
	for _, lr := range lendingResults {
		lendingMap[lr.AssetID] = lr.Total
	}

	// 2. Hitung jumlah yang rusak
	var rusakResults []qtyResult
	_ = r.db.WithContext(ctx).
		Table("asset_rusak").
		Select("asset_id, COALESCE(SUM(qty_rusak), 0) AS total").
		Where("asset_id IN ?", assetIDs).
		Group("asset_id").
		Scan(&rusakResults).Error

	rusakMap := make(map[uint]int, len(rusakResults))
	for _, rr := range rusakResults {
		rusakMap[rr.AssetID] = rr.Total
	}

	details := make([]AssetDetailInfo, len(assets))
	for i, a := range assets {
		qDipinjam := lendingMap[a.AssetID]
		qRusak := rusakMap[a.AssetID]
		qTersedia := a.QtyIn - (qDipinjam + qRusak)
		if qTersedia < 0 {
			qTersedia = 0
		}

		catName := "-"
		if a.Category.Name != "" {
			catName = a.Category.Name
		}
		locName := "-"
		if a.Location.Name != "" {
			locName = a.Location.Name
		}

		cond := a.Condition
		if cond == "" {
			cond = "Baik"
		}

		details[i] = AssetDetailInfo{
			AssetID:      a.AssetID,
			CodeAsset:    a.CodeAsset,
			NamaAsset:    a.NamaAsset,
			CategoryName: catName,
			LocationName: locName,
			Condition:    cond,
			Status:       a.Status,
			QtyTotal:     a.QtyIn,
			QtyRusak:     qRusak,
			QtyDipinjam:  qDipinjam,
			QtyTersedia:  qTersedia,
			Description:  a.Description,
		}
	}

	return details, nil
}
