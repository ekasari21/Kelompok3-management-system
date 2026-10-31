package textanalysis

import (
	"context"
	"testing"
	"time"
)

func TestAnalyzeValidation(t *testing.T) {
	s := NewService(nil)

	// Validasi source
	_, err := s.Analyze(context.Background(), "invalid_source", 3)
	if err == nil {
		t.Errorf("expected error for invalid source, got nil")
	}

	// Validasi clusters
	_, err = s.Analyze(context.Background(), "all", 1)
	if err == nil {
		t.Errorf("expected error for clusters < 2, got nil")
	}

	_, err = s.Analyze(context.Background(), "all", 11)
	if err == nil {
		t.Errorf("expected error for clusters > 10, got nil")
	}
}

func TestNativeGoTextClustering(t *testing.T) {
	svc := NewService(nil)

	// Sample data keluhan dan kerusakan aset
	sampleRecords := []TextRecord{
		{
			ID:          "1",
			Source:      "complaints",
			Title:       "Laptop Asus tidak mau menyala",
			Description: "Layar blank hitam dan laptop mati total saat tombol power ditekan",
			Quantity:    0,
			Status:      "pending",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "2",
			Source:      "damage",
			Title:       "Laptop Dell mati total",
			Description: "Adaptor charger laptop rusak dan laptop mati",
			Quantity:    2,
			Status:      "rusak_berat",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "3",
			Source:      "complaints",
			Title:       "AC Ruang Rapat bocor",
			Description: "AC mengeluarkan tetesan air dan tidak dingin sama sekali",
			Quantity:    0,
			Status:      "in_progress",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "4",
			Source:      "damage",
			Title:       "AC bocor dan bau",
			Description: "Unit AC pendingin ruangan bocor menetes ke meja",
			Quantity:    1,
			Status:      "rusak_sedang",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "5",
			Source:      "complaints",
			Title:       "Proyektor Epson mati mendadak",
			Description: "Lampu proyektor kedip merah dan gambar proyektor mati",
			Quantity:    0,
			Status:      "pending",
			CreatedAt:   time.Now(),
		},
	}

	// Jalankan clustering K-Means dengan k=3
	resp := svc.clusterRecords(sampleRecords, "all", 3)

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.TotalReports != len(sampleRecords) {
		t.Errorf("expected %d total reports, got %d", len(sampleRecords), resp.TotalReports)
	}

	if len(resp.Groups) == 0 {
		t.Fatal("expected groups to be formed, got 0")
	}

	// Pastikan grup memiliki kata kunci dan total unit rusak terakumulasi
	var totalDamagedUnits int64
	for _, g := range resp.Groups {
		if g.Name == "" {
			t.Errorf("cluster %d has empty name", g.ClusterID)
		}
		if g.ReportCount == 0 {
			t.Errorf("cluster %d has 0 reports", g.ClusterID)
		}
		totalDamagedUnits += g.TotalDamagedUnits
	}

	// Total quantity dari damage: ID 2 (2 unit) + ID 4 (1 unit) = 3 unit
	expectedDamageUnits := int64(3)
	if totalDamagedUnits != expectedDamageUnits {
		t.Errorf("expected %d total damaged units, got %d", expectedDamageUnits, totalDamagedUnits)
	}
}

func TestFallbackStopWordsOnly(t *testing.T) {
	svc := NewService(nil)

	records := []TextRecord{
		{
			ID:          "1",
			Source:      "complaints",
			Title:       "Ini itu ada",
			Description: "Yang dan di ke",
			Quantity:    0,
			Status:      "pending",
			CreatedAt:   time.Now(),
		},
	}

	resp := svc.clusterRecords(records, "all", 2)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if len(resp.Groups) != 1 {
		t.Errorf("expected 1 fallback group, got %d", len(resp.Groups))
	}

	if resp.Groups[0].Name != "Kelompok masalah umum" {
		t.Errorf("expected 'Kelompok masalah umum', got %q", resp.Groups[0].Name)
	}
}
