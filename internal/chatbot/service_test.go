package chatbot

import (
	"context"
	"strings"
	"testing"
)

func TestChatbotInventoryRule(t *testing.T) {
	// Repository dengan db nil untuk unit testing logika service
	repo := NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	// 1. UJI PERTANYAAN DI LUAR INVENTARIS (HARUS DITOLAK SESUAI ATURAN)
	nonInventoryQuestions := []string{
		"Siapa presiden Indonesia saat ini?",
		"Bagaimana cuaca di Jakarta hari ini?",
		"Bikinkan resep nasi goreng spesial",
		"Berapa hasil dari 25 dikali 14?",
		"Ceritakan lelucon lucu dong",
		"Buatkan script python untuk machine learning",
		"Siapa pemenang piala dunia 2022?",
		"Bagaimana ramalan bintang zodiak gemini?",
		"Kamu punya pacar gak?",
	}

	for _, q := range nonInventoryQuestions {
		resp, err := svc.ProcessMessage(ctx, q)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", q, err)
		}

		if resp.Intent != string(IntentOutOfDomain) {
			t.Errorf("pertanyaan di luar inventaris %q seharusnya menghasilkan IntentOutOfDomain, didapat: %s", q, resp.Intent)
		}

		if !strings.Contains(resp.Reply, "hanya diperbolehkan merespons dan menjawab pertanyaan seputar data inventaris") {
			t.Errorf("jawaban untuk %q harus menegaskan aturan inventaris, didapat: %s", q, resp.Reply)
		}
	}

	// 2. UJI PERTANYAAN SEPUTAR DATA INVENTARIS (HARUS DITERIMA DAN DIPROSES)
	inventoryTests := []struct {
		question       string
		expectedIntent Intent
	}{
		{"Halo selamat pagi", IntentGreeting},
		{"Terima kasih infonya", IntentThanks},
		{"Bantuan apa saja yang bisa ditanyakan?", IntentHelp},
		{"Berapa total aset sistem saat ini?", IntentSummary},
		{"Aset apa saja yang tersedia dan siap dipinjam?", IntentAvailable},
		{"Berapa banyak aset yang rusak berat?", IntentBroken},
		{"Siapa yang sedang meminjam aset?", IntentLending},
		{"Daftar lokasi penyimpanan barang apa saja?", IntentLocation},
		{"Apa saja kategori aset yang ada?", IntentCategory},
		{"Bagaimana cara pinjam aset di sistem?", IntentProcedure},
		{"Apakah ada laptop yang ready?", IntentAssetSearch},
		{"Cek stok proyektor", IntentAssetSearch},
		{"berap jumlah asset total", IntentSummary},
		{"berapa total asset", IntentSummary},
		{"jumlah asset", IntentSummary},
		{"stok asset tersedia", IntentAvailable},
	}

	for _, tt := range inventoryTests {
		resp, err := svc.ProcessMessage(ctx, tt.question)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.question, err)
		}

		if resp.Intent != string(tt.expectedIntent) {
			t.Errorf("pertanyaan %q: diharapkan intent %s, didapat %s", tt.question, tt.expectedIntent, resp.Intent)
		}

		if resp.Reply == "" {
			t.Errorf("pertanyaan %q: jawaban tidak boleh kosong", tt.question)
		}
	}
}
