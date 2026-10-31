package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Intent string

const (
	IntentGreeting    Intent = "greeting"
	IntentThanks      Intent = "thanks"
	IntentHelp        Intent = "help"
	IntentSummary     Intent = "summary"
	IntentAvailable   Intent = "available"
	IntentBroken      Intent = "broken"
	IntentLending     Intent = "lending"
	IntentLocation    Intent = "location"
	IntentCategory    Intent = "category"
	IntentAssetSearch Intent = "asset_search"
	IntentProcedure   Intent = "procedure"
	IntentComplaint   Intent = "complaint"
	IntentOutOfDomain Intent = "out_of_domain"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ProcessMessage memproses pesan pengguna dan menghasilkan jawaban yang HANYA seputar data inventaris.
func (s *Service) ProcessMessage(ctx context.Context, rawMessage string) (*ChatResponse, error) {
	msg := strings.TrimSpace(rawMessage)
	if msg == "" {
		return &ChatResponse{
			Success:     true,
			Reply:       "Silakan ketikkan pertanyaan seputar data inventaris yang ingin Anda ketahui.",
			Intent:      string(IntentHelp),
			Suggestions: s.GetInitialSuggestions(),
			Timestamp:   time.Now(),
		}, nil
	}

	// 1. Cek apakah pengguna memiliki GEMINI_API_KEY untuk pemrosesan berbasis LLM
	geminiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if geminiKey != "" {
		llmReply, err := s.tryGeminiAPI(ctx, geminiKey, msg)
		if err == nil && llmReply != "" {
			return &ChatResponse{
				Success:     true,
				Reply:       llmReply,
				Intent:      "gemini_assistant",
				Suggestions: s.getRelevantSuggestions(msg),
				Timestamp:   time.Now(),
			}, nil
		}
	}

	// 2. Jalankan Natural Language Engine Go deterministik berbasis database riil
	intent, entity := s.classifyQuery(msg)

	var reply string
	var suggestions []string

	switch intent {
	case IntentOutOfDomain:
		reply = "Mohon maaf, saya adalah Asisten Khusus Data Inventaris.\n\n" +
			"Sesuai aturan sistem, saya **hanya diperbolehkan merespons dan menjawab pertanyaan seputar data inventaris** " +
			"(seperti ketersediaan stok, peminjaman aset, kondisi kerusakan aset, lokasi ruangan, kategori barang, dan SOP inventaris).\n\n" +
			"Silakan ajukan pertanyaan yang berkaitan dengan data inventaris kami."
		suggestions = s.GetInitialSuggestions()

	case IntentGreeting:
		reply = "Halo! Selamat datang di Layanan Asisten Inventaris Cerdas.\n\n" +
			"Saya siap membantu Anda memperoleh informasi akurat seputar **data inventaris** sistem " +
			"(stok aset, ketersediaan, peminjaman, aset rusak, kategori, dan lokasi penyimpanan barang).\n\n" +
			"Ada data inventaris yang ingin Anda tanyakan hari ini?"
		suggestions = []string{
			"Berapa jumlah total aset?",
			"Aset apa saja yang tersedia?",
			"Berapa jumlah aset rusak?",
		}

	case IntentThanks:
		reply = "Sama-sama! Senang bisa membantu Anda mengelola data inventaris dengan lebih mudah.\n\n" +
			"Jika sewaktu-waktu membutuhkan informasi aset atau peminjaman lainnya, silakan tanyakan kembali."
		suggestions = s.GetInitialSuggestions()

	case IntentHelp:
		reply = "📋 **Topik Data Inventaris yang Dapat Saya Bantu:**\n\n" +
			"1. 📦 **Total & Rekap Aset:** 'Berapa total aset sistem?' / 'Jumlah keseluruhan barang'\n" +
			"2. ✅ **Stok & Ketersediaan:** 'Aset apa saja yang siap dipinjam?' / 'Stok laptop sisa berapa?'\n" +
			"3. 📋 **Peminjaman Aktif:** 'Siapa saja yang sedang pinjam aset?' / 'Daftar peminjaman'\n" +
			"4. 🔧 **Aset Rusak:** 'Berapa aset rusak?' / 'Aset rusak apa saja yang bisa diperbaiki?'\n" +
			"5. 🏢 **Lokasi Ruangan:** 'Daftar lokasi aset' / 'Aset di Ruang Server'\n" +
			"6. 🏷️ **Kategori:** 'Daftar kategori aset'\n" +
			"7. 🔍 **Pencarian Spesifik:** 'Cek proyektor' / 'Cari kode AST-001'\n" +
			"8. 📝 **Prosedur SOP:** 'Bagaimana cara pinjam aset?' / 'Cara lapor aset rusak'\n\n" +
			"*(Catatan: Sesuai aturan, saya hanya merespons pertanyaan terkait data inventaris)*"
		suggestions = s.GetInitialSuggestions()

	case IntentSummary:
		summary, err := s.repo.GetInventorySummary(ctx)
		if err != nil {
			return nil, err
		}
		reply = fmt.Sprintf("📊 **Ringkasan Data Inventaris Terkini:**\n\n"+
			"• **Total Seluruh Unit Aset:** %d unit (%d jenis aset aktif)\n"+
			"• **Aset Tersedia:** %d unit (siap digunakan/dipinjam)\n"+
			"• **Aset Sedang Dipinjam:** %d unit\n"+
			"• **Aset Rusak:** %d unit\n"+
			"   - Bisa diperbaiki: %d unit\n"+
			"   - Rusak berat/tidak bisa diperbaiki: %d unit\n"+
			"• **Master Data:** %d Kategori & %d Lokasi Ruangan.\n\n"+
			"Seluruh data di atas tersinkronisasi secara real-time dengan database sistem.",
			summary.TotalUnitAset, summary.TotalJenisAset,
			summary.AsetTersedia,
			summary.AsetDipinjam,
			summary.AsetRusak,
			summary.RusakBisaPerbaiki, summary.RusakTidakBisa,
			summary.TotalKategori, summary.TotalLokasi,
		)
		suggestions = []string{
			"Aset apa saja yang tersedia?",
			"Aset apa saja yang sedang dipinjam?",
			"Rincian aset rusak",
		}

	case IntentAvailable:
		availableList, err := s.repo.GetAvailableAssets(ctx, 8)
		if err != nil {
			return nil, err
		}
		if len(availableList) == 0 {
			reply = "Saat ini tidak ada unit aset yang tersedia (stok kosong atau seluruhnya sedang dipinjam/rusak)."
		} else {
			var sb strings.Builder
			sb.WriteString("✅ **Daftar Aset Tersedia (Siap Digunakan/Dipinjam):**\n\n")
			for i, a := range availableList {
				sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`)\n   - Kategori: %s | Lokasi: %s\n   - Sisa Stok Tersedia: **%d unit** (Total: %d, Pinjam: %d, Rusak: %d)\n",
					i+1, a.NamaAsset, a.CodeAsset, a.CategoryName, a.LocationName, a.QtyTersedia, a.QtyTotal, a.QtyDipinjam, a.QtyRusak))
			}
			sb.WriteString("\nUntuk mengajukan peminjaman, gunakan menu **Pinjam Asset**.")
			reply = sb.String()
		}
		suggestions = []string{
			"Bagaimana cara pinjam aset?",
			"Siapa yang sedang meminjam barang?",
			"Berapa total aset?",
		}

	case IntentBroken:
		filter := ""
		lowerMsg := strings.ToLower(msg)
		if strings.Contains(lowerMsg, "berat") {
			filter = "berat"
		} else if strings.Contains(lowerMsg, "ringan") {
			filter = "ringan"
		} else if strings.Contains(lowerMsg, "sedang") {
			filter = "sedang"
		}

		brokenList, err := s.repo.GetBrokenAssets(ctx, filter, 8)
		if err != nil {
			return nil, err
		}

		if len(brokenList) == 0 {
			reply = "Alhamdulillah, saat ini tidak ada catatan aset rusak yang sesuai dengan pencarian Anda."
		} else {
			var sb strings.Builder
			sb.WriteString("🔧 **Data Aset Rusak & Status Penanganan:**\n\n")
			for i, b := range brokenList {
				repairStatus := "Dapat Diperbaiki"
				if !b.IsRepaired {
					repairStatus = "Menunggu Tindak Lanjut"
				}
				sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`) - Kode Laporan: `%s`\n   - Jumlah Rusak: **%d unit** | Tingkat: *%s*\n   - Status Perbaikan: %s\n   - Keterangan: %s\n",
					i+1, b.NamaAsset, b.CodeAsset, b.CodeRusak, b.QtyRusak, strings.ReplaceAll(b.Status, "_", " "), repairStatus, b.Description))
			}
			sb.WriteString("\n💡 **Rekomendasi Penanganan:**\n" +
				"• Unit rusak ringan/sedang dapat segera dijadwalkan servis melalui teknisi.\n" +
				"• Unit rusak berat yang tidak dapat diperbaiki disarankan untuk diajukan penghapusan aset (write-off).")
			reply = sb.String()
		}
		suggestions = []string{
			"Aset rusak yang bisa diperbaiki",
			"Berapa jumlah total aset?",
			"Aset apa saja yang tersedia?",
		}

	case IntentLending:
		lendings, err := s.repo.GetActiveLendings(ctx, 8)
		if err != nil {
			return nil, err
		}

		if len(lendings) == 0 {
			reply = "Saat ini tidak ada aset yang sedang dipinjam (seluruh unit peminjaman telah dikembalikan)."
		} else {
			var sb strings.Builder
			sb.WriteString("📋 **Daftar Peminjaman Aset yang Sedang Aktif:**\n\n")
			for i, l := range lendings {
				sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`)\n   - Peminjam: **%s**\n   - Jumlah: %d unit | Status: *%s*\n",
					i+1, l.NamaAsset, l.CodeAsset, l.Borrower, l.Qty, l.Status))
			}
			sb.WriteString("\nPastikan pengembalian aset dikonfirmasi tepat waktu oleh Admin/Teknisi.")
			reply = sb.String()
		}
		suggestions = []string{
			"Berapa aset yang tersedia?",
			"Berapa total aset?",
			"Cara pengembalian aset",
		}

	case IntentLocation:
		if entity != "" {
			locAssets, err := s.repo.GetAssetsByLocation(ctx, entity, 8)
			if err != nil {
				return nil, err
			}
			if len(locAssets) == 0 {
				reply = fmt.Sprintf("Tidak ditemukan data aset pada lokasi/ruangan '%s'. Silakan periksa daftar lokasi yang terdaftar di sistem.", entity)
			} else {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("🏢 **Daftar Aset di Lokasi '%s':**\n\n", locAssets[0].LocationName))
				for i, a := range locAssets {
					sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`)\n   - Total: %d unit | Tersedia: %d unit | Kondisi: %s\n",
						i+1, a.NamaAsset, a.CodeAsset, a.QtyTotal, a.QtyTersedia, a.Condition))
				}
				reply = sb.String()
			}
		} else {
			locations, err := s.repo.GetLocationsWithCount(ctx)
			if err != nil {
				return nil, err
			}
			if len(locations) == 0 {
				reply = "Belum ada data lokasi atau ruangan yang terdaftar di master data lokasi."
			} else {
				var sb strings.Builder
				sb.WriteString("🏢 **Daftar Lokasi Penyimpanan Aset Terdaftar:**\n\n")
				for i, loc := range locations {
					desc := loc.Description
					if desc == "" {
						desc = "-"
					}
					sb.WriteString(fmt.Sprintf("%d. **%s** (%d aset tersimpan)\n   - Keterangan: %s\n",
						i+1, loc.Name, loc.AssetCount, desc))
				}
				sb.WriteString("\n*Tips: Anda dapat menanyakan 'Aset di [Nama Ruangan]' untuk melihat rincian isi ruangan tertentu.*")
				reply = sb.String()
			}
		}
		suggestions = []string{
			"Berapa total aset?",
			"Aset apa saja yang tersedia?",
			"Daftar kategori aset",
		}

	case IntentCategory:
		if entity != "" {
			catAssets, err := s.repo.GetAssetsByCategory(ctx, entity, 8)
			if err != nil {
				return nil, err
			}
			if len(catAssets) == 0 {
				reply = fmt.Sprintf("Tidak ditemukan data aset pada kategori '%s'.", entity)
			} else {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("🏷️ **Daftar Aset dalam Kategori '%s':**\n\n", catAssets[0].CategoryName))
				for i, a := range catAssets {
					sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`)\n   - Total: %d unit | Tersedia: %d unit | Lokasi: %s\n",
						i+1, a.NamaAsset, a.CodeAsset, a.QtyTotal, a.QtyTersedia, a.LocationName))
				}
				reply = sb.String()
			}
		} else {
			categories, err := s.repo.GetCategoriesWithCount(ctx)
			if err != nil {
				return nil, err
			}
			if len(categories) == 0 {
				reply = "Belum ada kategori yang terdaftar di master data kategori aset."
			} else {
				var sb strings.Builder
				sb.WriteString("🏷️ **Daftar Master Kategori Aset Terdaftar:**\n\n")
				for i, cat := range categories {
					sb.WriteString(fmt.Sprintf("%d. **%s** (%d aset)\n", i+1, cat.Name, cat.AssetCount))
				}
				sb.WriteString("\n*Tips: Tanyakan 'Aset kategori [Nama Kategori]' untuk melihat detail barang pada kategori tersebut.*")
				reply = sb.String()
			}
		}
		suggestions = []string{
			"Berapa total aset?",
			"Aset apa saja yang tersedia?",
			"Daftar lokasi aset",
		}

	case IntentAssetSearch:
		searchTerm := entity
		if searchTerm == "" {
			searchTerm = msg
		}
		assets, err := s.repo.SearchAssets(ctx, searchTerm, 5)
		if err != nil {
			return nil, err
		}

		if len(assets) == 0 {
			reply = fmt.Sprintf("Aset dengan kata kunci '**%s**' tidak ditemukan di database inventaris kami.\n\n"+
				"Pastikan nama barang atau kode aset sudah sesuai, atau tanyakan 'Aset apa saja yang tersedia?' untuk melihat daftar barang yang ready.", searchTerm)
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("🔍 **Hasil Pencarian Data Aset ('%s'):**\n\n", searchTerm))
			for i, a := range assets {
				sb.WriteString(fmt.Sprintf("%d. **%s** (`%s`)\n", i+1, a.NamaAsset, a.CodeAsset))
				sb.WriteString(fmt.Sprintf("   • Kategori: %s\n", a.CategoryName))
				sb.WriteString(fmt.Sprintf("   • Lokasi: %s\n", a.LocationName))
				sb.WriteString(fmt.Sprintf("   • Kondisi: %s\n", a.Condition))
				sb.WriteString(fmt.Sprintf("   • Total Unit: %d unit\n", a.QtyTotal))
				sb.WriteString(fmt.Sprintf("   • **Tersedia Siap Pakai:** **%d unit**\n", a.QtyTersedia))
				sb.WriteString(fmt.Sprintf("   • Sedang Dipinjam: %d unit | Rusak: %d unit\n", a.QtyDipinjam, a.QtyRusak))
				if a.Description != "" {
					sb.WriteString(fmt.Sprintf("   • Deskripsi: %s\n", a.Description))
				}
				sb.WriteString("\n")
			}
			reply = strings.TrimSpace(sb.String())
		}
		suggestions = []string{
			"Berapa total aset?",
			"Aset apa saja yang tersedia?",
			"Bagaimana cara pinjam aset?",
		}

	case IntentProcedure:
		lowerMsg := strings.ToLower(msg)
		if strings.Contains(lowerMsg, "pinjam") {
			reply = "📋 **Alur & Prosedur Peminjaman Aset:**\n\n" +
				"1. Masuk ke panel akun Anda dan buka menu **Pinjam Asset**.\n" +
				"2. Pilih unit barang yang berstatus **Tersedia** dan tentukan jumlah unit yang dibutuhkan.\n" +
				"3. Masukkan tanggal pinjam dan estimasi tanggal pengembalian.\n" +
				"4. Kirim pengajuan peminjaman. Admin atau Teknisi akan memeriksa persetujuan barang.\n" +
				"5. Setelah selesai digunakan, kembalikan barang tepat waktu agar dicatat statusnya menjadi 'dikembalikan'."
		} else if strings.Contains(lowerMsg, "rusak") || strings.Contains(lowerMsg, "keluhan") || strings.Contains(lowerMsg, "komplain") {
			reply = "🔧 **Alur & Prosedur Pelaporan Kerusakan / Komplain Aset:**\n\n" +
				"1. Buka menu **Lapor Keluhan** pada antarmuka akun Anda.\n" +
				"2. Pilih aset yang mengalami masalah atau kerusakan.\n" +
				"3. Unggah foto bukti fisik kerusakan barang.\n" +
				"4. Berikan judul dan deskripsi detail terkait kendala yang dialami.\n" +
				"5. Tentukan tingkat urgensi (Low, Medium, High, Critical).\n" +
				"6. Teknisi akan menerima laporan dan melakukan verifikasi penanganan pada menu Kelola Aset Rusak."
		} else {
			reply = "📖 **Panduan Prosedur Sistem Inventaris:**\n\n" +
				"• **Peminjaman:** Ajukan melalui menu *Pinjam Asset*, pilih aset ready dan tentukan durasi pinjam.\n" +
				"• **Pelaporan Kerusakan:** Kirim laporan beserta foto melalui menu *Lapor Keluhan*.\n" +
				"• **Pengembalian:** Dilakukan melalui konfirmasi Teknisi/Admin saat aset dikembalikan secara fisik."
		}
		suggestions = []string{
			"Aset apa saja yang tersedia?",
			"Siapa yang sedang meminjam barang?",
			"Berapa total aset?",
		}

	case IntentComplaint:
		reply = "📢 **Informasi Laporan Keluhan & Kerusakan Aset:**\n\n" +
			"Seluruh laporan komplain terkait fasilitas dan aset inventaris dikelola pada menu **Kelola Keluhan**.\n" +
			"Sistem juga dilengkapi dengan analisis Machine Learning klasterisasi otomatis (TF-IDF & K-Means) " +
			"untuk mengelompokkan keluhan serupa dan mempercepat tindakan teknisi."
		suggestions = []string{
			"Berapa jumlah aset rusak?",
			"Aset apa saja yang tersedia?",
			"Berapa total aset?",
		}

	default:
		reply = "Mohon maaf, saya belum memahami pertanyaan tersebut. " +
			"Saya adalah Asisten Khusus Inventaris yang hanya melayani pertanyaan seputar stok aset, ketersediaan, " +
			"peminjaman, aset rusak, kategori, dan lokasi ruangan."
		suggestions = s.GetInitialSuggestions()
	}

	return &ChatResponse{
		Success:     true,
		Reply:       reply,
		Intent:      string(intent),
		Suggestions: suggestions,
		Timestamp:   time.Now(),
	}, nil
}

// classifyQuery mengklasifikasikan pertanyaan dan mengecek apakah terkait data inventaris atau di luar domain.
func (s *Service) classifyQuery(q string) (Intent, string) {
	norm := strings.ToLower(strings.TrimSpace(q))

	// Hapus tanda baca di awal/akhir
	norm = regexp.MustCompile(`^[^\w]+|[^\w]+$`).ReplaceAllString(norm, "")

	// ========================================================
	// 1. CEK OUT-OF-DOMAIN SECARA KETAT (NON-INVENTARIS BLACKLIST)
	// ========================================================
	outOfDomainPatterns := []string{
		// Politik & Tokoh Pemerintahan Umum
		"presiden", "menteri", "partai", "pemilu", "pilpres", "pilkada", "dpr", "gubernur", "bupati", "politik",
		// Cuaca & Alam
		"cuaca", "ramalan cuaca", "hujan", "gempa", "angin topan", "suhu hari ini", "panas sekali",
		// Masakan & Resep Makanan
		"resep", "cara masak", "bumbu", "makanan", "kuliner", "nasi goreng", "martabak", "gorengan",
		// Pengetahuan Umum Luar / Matematika / Sains Murni
		"hitung", "kalkulator", "akar kuadrat", "rumus fisika", "biologi", "sejarah indonesia", "ibukota",
		// Hiburan, Film, Musik, Humor, Zodiak
		"lelucon", "lawak", "cerita lucu", "dongeng", "film", "lagu", "lirik", "artis", "aktor", "bioskop",
		"zodiak", "ramalan bintang", "game", "sepak bola", "piala dunia",
		// Pemrograman & Coding Umum di luar fungsi sistem
		"coding python", "buatkan script", "belajar javascript", "algoritma sorting", "buat fungsi",
		// Percakapan Pribadi / Curhat
		"curhat", "pacar", "cinta", "jodoh", "usia kamu", "umur kamu", "punya pacar",
	}

	for _, p := range outOfDomainPatterns {
		if strings.Contains(norm, p) {
			// Pastikan bukan keyword aset ganda
			if !strings.Contains(norm, "aset") && !strings.Contains(norm, "inventaris") && !strings.Contains(norm, "barang") {
				return IntentOutOfDomain, ""
			}
		}
	}

	// ========================================================
	// 2. CEK SALAM, TERIMA KASIH, BANTUAN
	// ========================================================
	greetings := []string{"halo", "hai", "selamat pagi", "selamat siang", "selamat sore", "selamat malam", "assalamualaikum", "assalamu'alaikum", "hi", "hey", "ping", "pagi", "siang", "sore", "malam", "tes", "test"}
	for _, g := range greetings {
		if norm == g || strings.HasPrefix(norm, g+" ") || strings.HasSuffix(norm, " "+g) {
			return IntentGreeting, ""
		}
	}

	thanks := []string{"terima kasih", "makasih", "thanks", "terimakasih", "tengkyu", "sukron", "ok terima kasih", "oke makasih"}
	for _, t := range thanks {
		if strings.Contains(norm, t) {
			return IntentThanks, ""
		}
	}

	helps := []string{"bantuan", "help", "menu", "apa yang bisa kamu lakukan", "bisa tanya apa", "contoh pertanyaan", "panduan bot", "kemampuan"}
	for _, h := range helps {
		if strings.Contains(norm, h) {
			return IntentHelp, ""
		}
	}

	// ========================================================
	// 3. PENCARIAN ASET SPESIFIK BERDASARKAN NAMA BARANG / KODE
	// ========================================================
	itemNouns := []string{
		"laptop", "komputer", "pc", "proyektor", "infocus", "monitor", "meja", "kursi",
		"printer", "scanner", "kamera", "ac", "air conditioner", "speaker", "mouse",
		"keyboard", "router", "switch", "server", "kabel", "lemari", "rak", "whiteboard",
		"papan tulis", "dispenser",
	}

	for _, item := range itemNouns {
		if strings.Contains(norm, item) {
			return IntentAssetSearch, item
		}
	}

	// Cek kode aset (seperti AST-..., AST/..., dll)
	codePattern := regexp.MustCompile(`ast[-\d\w]+|[a-z]{3}-\d+`)
	if match := codePattern.FindString(norm); match != "" {
		return IntentAssetSearch, match
	}

	// Cek awalan perintah pencarian ("cari proyektor", "cek laptop", dsb)
	searchPrefixes := []string{"cari ", "cek ", "stok ", "detail ", "informasi ", "apakah ada "}
	for _, prefix := range searchPrefixes {
		if strings.HasPrefix(norm, prefix) {
			extracted := strings.TrimSpace(strings.TrimPrefix(norm, prefix))
			if len(extracted) >= 3 && !strings.Contains(extracted, "aset") {
				return IntentAssetSearch, extracted
			}
		}
	}

	// ========================================================
	// 4. INTEN PROSEDUR / SOP
	// ========================================================
	if strings.Contains(norm, "cara") || strings.Contains(norm, "alur") ||
		strings.Contains(norm, "prosedur") || strings.Contains(norm, "sop") ||
		strings.Contains(norm, "bagaimana") && (strings.Contains(norm, "pinjam") || strings.Contains(norm, "lapor")) {
		return IntentProcedure, ""
	}

	// ========================================================
	// 5. INTEN ASET RUSAK / KERUSAKAN / PERBAIKAN
	// ========================================================
	if strings.Contains(norm, "rusak") || strings.Contains(norm, "kerusakan") ||
		strings.Contains(norm, "perbaikan") || strings.Contains(norm, "reparasi") ||
		strings.Contains(norm, "servis") || strings.Contains(norm, "service") {
		return IntentBroken, ""
	}

	// ========================================================
	// 6. INTEN ASET TERSEDIA / STOK SIAP PAKAI
	// ========================================================
	if strings.Contains(norm, "tersedia") || strings.Contains(norm, "ready") ||
		strings.Contains(norm, "siap pakai") || strings.Contains(norm, "sisa stok") ||
		strings.Contains(norm, "stok ada") {
		return IntentAvailable, ""
	}

	// ========================================================
	// 7. INTEN PEMINJAMAN / LENDING
	// ========================================================
	if strings.Contains(norm, "pinjam") || strings.Contains(norm, "minjam") ||
		strings.Contains(norm, "meminjam") || strings.Contains(norm, "dipinjam") ||
		strings.Contains(norm, "peminjam") || strings.Contains(norm, "peminjaman") ||
		strings.Contains(norm, "lending") || strings.Contains(norm, "terlambat") {
		return IntentLending, ""
	}

	// ========================================================
	// 8. INTEN LOKASI & RUANGAN
	// ========================================================
	if strings.Contains(norm, "lokasi") || strings.Contains(norm, "ruang") ||
		strings.Contains(norm, "ruangan") || strings.Contains(norm, "gedung") ||
		strings.Contains(norm, "lab") || strings.Contains(norm, "gudang") ||
		strings.Contains(norm, "disimpan di") || strings.Contains(norm, "penempatan") {
		entity := s.extractLocationEntity(norm)
		return IntentLocation, entity
	}

	// ========================================================
	// 9. INTEN KATEGORI
	// ========================================================
	if strings.Contains(norm, "kategori") || strings.Contains(norm, "jenis aset") ||
		strings.Contains(norm, "jenis barang") || strings.Contains(norm, "golongan") {
		entity := s.extractCategoryEntity(norm)
		return IntentCategory, entity
	}

	// ========================================================
	// 10. INTEN KELUHAN / KOMPLAIN
	// ========================================================
	if strings.Contains(norm, "keluhan") || strings.Contains(norm, "komplain") ||
		strings.Contains(norm, "laporan keluhan") || strings.Contains(norm, "complaint") ||
		strings.Contains(norm, "aduan") {
		return IntentComplaint, ""
	}

	// ========================================================
	// 11. INTEN TOTAL / RINGKASAN / REKAPITULASI ASET
	// ========================================================
	if strings.Contains(norm, "total") || strings.Contains(norm, "jumlah") ||
		strings.Contains(norm, "semua aset") || strings.Contains(norm, "banyak aset") ||
		strings.Contains(norm, "keseluruhan") || strings.Contains(norm, "rekap") ||
		strings.Contains(norm, "statistik") || strings.Contains(norm, "ringkasan") ||
		strings.Contains(norm, "aset") || strings.Contains(norm, "barang") ||
		strings.Contains(norm, "inventaris") || strings.Contains(norm, "unit") {
		return IntentSummary, ""
	}

	// ========================================================
	// 12. APABILA TIDAK ADA KORELASI DENGAN INVENTARIS -> REJECT!
	// ========================================================
	return IntentOutOfDomain, ""
}

// extractLocationEntity mengambil kata nama ruangan bila disebutkan dalam pertanyaan.
func (s *Service) extractLocationEntity(q string) string {
	patterns := []string{
		`ruang\s+([a-zA-Z0-9\s]+)`,
		`ruangan\s+([a-zA-Z0-9\s]+)`,
		`gedung\s+([a-zA-Z0-9\s]+)`,
		`lab\s+([a-zA-Z0-9\s]+)`,
		`gudang`,
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p)
		matches := re.FindStringSubmatch(q)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		} else if len(matches) == 1 {
			return strings.TrimSpace(matches[0])
		}
	}

	return ""
}

// extractCategoryEntity mengambil kata kategori bila disebutkan dalam pertanyaan.
func (s *Service) extractCategoryEntity(q string) string {
	patterns := []string{
		`kategori\s+([a-zA-Z0-9\s]+)`,
		`elektronik`,
		`furnitur`,
		`atk`,
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p)
		matches := re.FindStringSubmatch(q)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		} else if len(matches) == 1 {
			return strings.TrimSpace(matches[0])
		}
	}

	return ""
}

// GetInitialSuggestions menyediakan tombol pertanyaan cepat di antarmuka chatbot.
func (s *Service) GetInitialSuggestions() []string {
	return []string{
		"Berapa jumlah total aset?",
		"Aset apa saja yang tersedia?",
		"Berapa jumlah aset rusak?",
		"Siapa yang sedang meminjam barang?",
		"Di mana saja lokasi penyimpanan aset?",
		"Bagaimana alur peminjaman aset?",
	}
}

// getRelevantSuggestions memberikan saran kontekstual berdasarkan pertanyaan.
func (s *Service) getRelevantSuggestions(q string) []string {
	norm := strings.ToLower(q)
	if strings.Contains(norm, "pinjam") {
		return []string{
			"Aset apa saja yang tersedia?",
			"Siapa saja peminjam aktif?",
			"Bagaimana cara pinjam aset?",
		}
	}
	if strings.Contains(norm, "rusak") {
		return []string{
			"Aset rusak yang bisa diperbaiki",
			"Bagaimana cara lapor aset rusak?",
			"Berapa total aset?",
		}
	}
	return s.GetInitialSuggestions()
}

// tryGeminiAPI memanggil endpoint Gemini REST API jika GEMINI_API_KEY disetel di sistem.
func (s *Service) tryGeminiAPI(ctx context.Context, apiKey, userMessage string) (string, error) {
	summary, err := s.repo.GetInventorySummary(ctx)
	if err != nil {
		summary = &InventorySummary{}
	}

	sysInstruction := fmt.Sprintf(
		"Kamu adalah Asisten Khusus Data Inventaris pada Sistem Manajemen Inventaris & Aset (Kelompok 3).\n\n"+
			"ATURAN MUTLAK:\n"+
			"1. Kamu HANYA DAN HANYA BOLEH menjawab pertanyaan seputar data inventaris (stok aset, ketersediaan, peminjaman barang, aset rusak, kategori, lokasi ruangan, dan prosedur SOP inventaris).\n"+
			"2. JIKA pengguna menanyakan apapun di luar topik data inventaris (seperti politik, cuaca, hiburan, resep makanan, coding tutorial umum, lelucon, atau pengetahuan umum selain aset sistem), kamu WAJIB MENOLAK secara sopan dan tegas dengan jawaban: 'Mohon maaf, saya adalah Asisten Khusus Data Inventaris. Sesuai aturan sistem, saya hanya dapat menjawab pertanyaan seputar data inventaris (seperti ketersediaan aset, stok barang, peminjaman, aset rusak, kategori, dan lokasi ruangan). Silakan ajukan pertanyaan terkait data inventaris.'\n"+
			"3. Fakta inventaris terkini saat ini:\n"+
			"   - Total unit aset: %d unit (%d jenis aset aktif)\n"+
			"   - Aset tersedia: %d unit\n"+
			"   - Aset sedang dipinjam: %d unit\n"+
			"   - Aset rusak: %d unit (%d unit bisa diperbaiki, %d unit rusak berat)\n"+
			"   - Total Kategori: %d, Total Lokasi: %d.\n"+
			"Jawablah dengan bahasa Indonesia yang ramah, profesional, ringkas, dan jelas menggunakan format Markdown.",
		summary.TotalUnitAset, summary.TotalJenisAset,
		summary.AsetTersedia,
		summary.AsetDipinjam,
		summary.AsetRusak,
		summary.RusakBisaPerbaiki, summary.RusakTidakBisa,
		summary.TotalKategori, summary.TotalLokasi,
	)

	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": sysInstruction},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": userMessage},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 600,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini api returned status %d", resp.StatusCode)
	}

	var resData struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return "", err
	}

	if len(resData.Candidates) > 0 && len(resData.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(resData.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("empty response from gemini")
}
