package chatbot

import "time"

// ChatRequest mewakili payload request pertanyaan pengguna.
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// ChatResponse mewakili format respons jawaban dari chatbot.
type ChatResponse struct {
	Success     bool      `json:"success"`
	Reply       string    `json:"reply"`
	Intent      string    `json:"intent,omitempty"`
	Suggestions []string  `json:"suggestions,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// InventorySummary menampung data statistik ringkasan inventaris untuk chatbot.
type InventorySummary struct {
	TotalJenisAset    int64 `json:"total_jenis_aset"`
	TotalUnitAset     int64 `json:"total_unit_aset"`
	AsetTersedia      int64 `json:"aset_tersedia"`
	AsetDipinjam      int64 `json:"aset_dipinjam"`
	AsetRusak         int64 `json:"aset_rusak"`
	RusakBisaPerbaiki int64 `json:"rusak_bisa_perbaiki"`
	RusakTidakBisa    int64 `json:"rusak_tidak_bisa"`
	TotalKategori     int64 `json:"total_kategori"`
	TotalLokasi       int64 `json:"total_lokasi"`
}

// AssetDetailInfo menampung data detail suatu aset beserta ketersediaannya.
type AssetDetailInfo struct {
	AssetID      uint   `json:"asset_id"`
	CodeAsset    string `json:"code_asset"`
	NamaAsset    string `json:"nama_asset"`
	CategoryName string `json:"category_name"`
	LocationName string `json:"location_name"`
	Condition    string `json:"condition"`
	Status       bool   `json:"status"`
	QtyTotal     int    `json:"qty_total"`
	QtyRusak     int    `json:"qty_rusak"`
	QtyDipinjam  int    `json:"qty_dipinjam"`
	QtyTersedia  int    `json:"qty_tersedia"`
	Description  string `json:"description"`
}

// LendingInfo menampung data peminjaman aset yang sedang berjalan.
type LendingInfo struct {
	LendingID   uint       `json:"lending_id"`
	Borrower    string     `json:"borrower"`
	NamaAsset   string     `json:"nama_asset"`
	CodeAsset   string     `json:"code_asset"`
	Qty         int        `json:"qty"`
	LendingDate time.Time  `json:"lending_date"`
	ReturnDate  *time.Time `json:"return_date"`
	Status      string     `json:"status"`
}

// BrokenInfo menampung data aset yang sedang dalam status rusak.
type BrokenInfo struct {
	RusakID     uint   `json:"rusak_id"`
	CodeRusak   string `json:"code_rusak"`
	NamaAsset   string `json:"nama_asset"`
	CodeAsset   string `json:"code_asset"`
	QtyRusak    int    `json:"qty_rusak"`
	Status      string `json:"status"`
	IsRepaired  bool   `json:"is_repaired"`
	Description string `json:"description"`
}

// LocationItem menampung data lokasi beserta jumlah aset di dalamnya.
type LocationItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AssetCount  int64  `json:"asset_count"`
}

// CategoryItem menampung data kategori beserta jumlah aset di dalamnya.
type CategoryItem struct {
	Name       string `json:"name"`
	AssetCount int64  `json:"asset_count"`
}
