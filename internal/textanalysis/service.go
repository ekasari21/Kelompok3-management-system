package textanalysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var stopWordsID = map[string]bool{
	"yang": true, "dan": true, "di": true, "ke": true, "dari": true, "untuk": true, "pada": true,
	"dengan": true, "karena": true, "ini": true, "itu": true, "ada": true, "tidak": true,
	"sudah": true, "belum": true, "akan": true, "dalam": true, "oleh": true, "saat": true,
	"agar": true, "atau": true, "juga": true, "serta": true, "maka": true, "dapat": true,
	"bisa": true, "telah": true, "tersebut": true, "mengalami": true, "terjadi": true,
	"aset": true, "barang": true, "unit": true, "buah": true, "buahnya": true,
	"mohon": true, "tolong": true, "bapak": true, "ibu": true, "terima": true, "kasih": true,
	"nya": true, "saya": true, "kami": true, "kita": true, "adalah": true,
}

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

// Analyze menjalankan analisis teks, TF-IDF, dan K-Means clustering menggunakan bahasa Go murni.
func (s *Service) Analyze(
	ctx context.Context,
	source string,
	clusters int,
) (*AnalysisResponse, error) {

	// 1. Validasi parameter input
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "all"
	}

	if source != "all" && source != "complaints" && source != "damage" {
		return nil, fmt.Errorf("source harus all, complaints, atau damage")
	}

	if clusters < 2 || clusters > 10 {
		return nil, fmt.Errorf("clusters harus antara 2 sampai 10")
	}

	// 2. Ambil data laporan dari repository jika repository tersedia
	var records []TextRecord
	if s.repository != nil {
		var err error
		records, err = s.repository.GetRecords(ctx)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil data laporan: %w", err)
		}
	}

	// 3. Filter data berdasarkan sumber dan bersihkan record kosong
	filtered := make([]TextRecord, 0, len(records))
	for _, record := range records {
		if source != "all" && record.Source != source {
			continue
		}

		cleaned := cleanText(record.Title + " " + record.Description)
		if cleaned == "" {
			continue
		}

		filtered = append(filtered, record)
	}

	// 4. Jika data kosong, kembalikan respons kosong
	if len(filtered) == 0 {
		return &AnalysisResponse{
			Source:            source,
			TotalReports:      0,
			ClustersRequested: clusters,
			Groups:            []ClusterGroup{},
		}, nil
	}

	// 5. Eksekusi engine Text Analysis & K-Means murni Go
	return s.clusterRecords(filtered, source, clusters), nil
}

// clusterRecords melakukan tokenisasi n-gram, ekstraksi TF-IDF, dan pengelompokan K-Means murni di Go.
func (s *Service) clusterRecords(records []TextRecord, source string, requestedClusters int) *AnalysisResponse {
	n := len(records)
	texts := make([]string, n)
	docNGrams := make([][]string, n)

	uniqueTextMap := make(map[string]bool)
	for i, r := range records {
		clean := cleanText(r.Title + " " + r.Description)
		texts[i] = clean
		uniqueTextMap[clean] = true
		docNGrams[i] = extractNGrams(clean)
	}

	// Hitung Document Frequency (DF) untuk setiap n-gram
	dfMap := make(map[string]int)
	for _, ngrams := range docNGrams {
		seenInDoc := make(map[string]bool)
		for _, term := range ngrams {
			if !seenInDoc[term] {
				seenInDoc[term] = true
				dfMap[term]++
			}
		}
	}

	// Jika tidak ada n-gram yang valid (misal seluruh teks berisi stop words), satukan dalam 1 kelompok
	if len(dfMap) == 0 {
		return s.createFallbackGroup(records, source, requestedClusters)
	}

	// Pilih vocabulary (maksimal 500 fitur dengan DF tertinggi seperti max_features=500 scikit-learn)
	type termScore struct {
		term string
		df   int
	}
	var termList []termScore
	for term, df := range dfMap {
		termList = append(termList, termScore{term: term, df: df})
	}
	sort.Slice(termList, func(i, j int) bool {
		if termList[i].df == termList[j].df {
			return termList[i].term < termList[j].term
		}
		return termList[i].df > termList[j].df
	})

	maxFeatures := 500
	if len(termList) > maxFeatures {
		termList = termList[:maxFeatures]
	}

	vocab := make(map[string]int)
	featureTerms := make([]string, len(termList))
	for idx, ts := range termList {
		vocab[ts.term] = idx
		featureTerms[idx] = ts.term
	}

	dim := len(featureTerms)

	// Hitung Matrix TF-IDF (dengan sublinear TF dan smoothed IDF)
	matrix := make([][]float64, n)
	for i := 0; i < n; i++ {
		vec := make([]float64, dim)
		termCounts := make(map[string]int)
		for _, term := range docNGrams[i] {
			termCounts[term]++
		}

		var normSq float64
		for term, count := range termCounts {
			if colIdx, ok := vocab[term]; ok {
				// Sublinear TF: 1 + ln(count)
				tf := 1.0 + math.Log(float64(count))
				// Smoothed IDF: ln((1 + N) / (1 + DF)) + 1
				df := dfMap[term]
				idf := math.Log(float64(1+n)/float64(1+df)) + 1.0
				val := tf * idf
				vec[colIdx] = val
				normSq += val * val
			}
		}

		// L2-Normalize vector
		if normSq > 0 {
			norm := math.Sqrt(normSq)
			for j := 0; j < dim; j++ {
				vec[j] /= norm
			}
		}
		matrix[i] = vec
	}

	// Tentukan jumlah cluster aktual (K)
	k := requestedClusters
	if k > n {
		k = n
	}
	if k > len(uniqueTextMap) {
		k = len(uniqueTextMap)
	}
	if k < 1 {
		k = 1
	}

	// Jalankan K-Means
	assignments := runKMeans(matrix, k)

	// Kelompokkan hasil per cluster
	clusterMembers := make(map[int][]int)
	for docIdx, cID := range assignments {
		clusterMembers[cID] = append(clusterMembers[cID], docIdx)
	}

	var groups []ClusterGroup
	clusterCounter := 1

	for _, memberIndices := range clusterMembers {
		if len(memberIndices) == 0 {
			continue
		}

		// Hitung mean score TF-IDF pada cluster untuk menentukan kata kunci dominan
		meanScores := make([]float64, dim)
		for _, docIdx := range memberIndices {
			for j := 0; j < dim; j++ {
				meanScores[j] += matrix[docIdx][j]
			}
		}
		mCount := float64(len(memberIndices))
		for j := 0; j < dim; j++ {
			meanScores[j] /= mCount
		}

		// Urutkan kata kunci tertinggi
		type termRank struct {
			term  string
			score float64
		}
		var ranked []termRank
		for j := 0; j < dim; j++ {
			if meanScores[j] > 0 {
				ranked = append(ranked, termRank{term: featureTerms[j], score: meanScores[j]})
			}
		}
		sort.Slice(ranked, func(i, j int) bool {
			return ranked[i].score > ranked[j].score
		})

		var keywords []string
		for idx, r := range ranked {
			if idx >= 5 {
				break
			}
			keywords = append(keywords, r.term)
		}

		var items []ClusterItem
		var totalDamagedUnits int64

		for _, docIdx := range memberIndices {
			rec := records[docIdx]
			if rec.Source == "damage" {
				totalDamagedUnits += rec.Quantity
			}

			items = append(items, ClusterItem{
				ID:          rec.ID,
				Source:      rec.Source,
				Title:       rec.Title,
				Description: rec.Description,
				Quantity:    rec.Quantity,
				Status:      rec.Status,
				CreatedAt:   rec.CreatedAt,
			})
		}

		groups = append(groups, ClusterGroup{
			ClusterID:         clusterCounter,
			Name:              buildGroupName(keywords),
			Keywords:          keywords,
			ReportCount:       len(items),
			TotalDamagedUnits: totalDamagedUnits,
			Items:             items,
		})
		clusterCounter++
	}

	// Urutkan grup berdasarkan jumlah laporan terbanyak (descending)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ReportCount > groups[j].ReportCount
	})

	// Beri nomor cluster ID urut 1, 2, ...
	for idx := range groups {
		groups[idx].ClusterID = idx + 1
	}

	return &AnalysisResponse{
		Source:            source,
		TotalReports:      n,
		ClustersRequested: requestedClusters,
		Groups:            groups,
	}
}

// runKMeans mengelompokkan dokumen ke k centroid menggunakan K-Means++.
func runKMeans(matrix [][]float64, k int) []int {
	n := len(matrix)
	if n == 0 || k <= 0 {
		return nil
	}
	dim := len(matrix[0])

	assignments := make([]int, n)
	if k == 1 {
		return assignments
	}

	// 1. Inisialisasi centroid secara deterministik (K-Means++ furthest heuristic)
	centroids := make([][]float64, k)
	centroids[0] = make([]float64, dim)
	copy(centroids[0], matrix[0])

	for c := 1; c < k; c++ {
		maxDist := -1.0
		furthestIdx := 0
		for i := 0; i < n; i++ {
			minDistToPrev := math.MaxFloat64
			for prev := 0; prev < c; prev++ {
				d := euclideanDistSq(matrix[i], centroids[prev])
				if d < minDistToPrev {
					minDistToPrev = d
				}
			}
			if minDistToPrev > maxDist {
				maxDist = minDistToPrev
				furthestIdx = i
			}
		}
		centroids[c] = make([]float64, dim)
		copy(centroids[c], matrix[furthestIdx])
	}

	// 2. Iterasi Lloyd
	maxIter := 50
	for iter := 0; iter < maxIter; iter++ {
		changed := false

		// Assignment step
		for i := 0; i < n; i++ {
			bestCluster := 0
			bestDist := math.MaxFloat64
			for c := 0; c < k; c++ {
				d := euclideanDistSq(matrix[i], centroids[c])
				if d < bestDist {
					bestDist = d
					bestCluster = c
				}
			}
			if assignments[i] != bestCluster {
				assignments[i] = bestCluster
				changed = true
			}
		}

		if !changed && iter > 0 {
			break
		}

		// Centroid update step
		counts := make([]int, k)
		newCentroids := make([][]float64, k)
		for c := 0; c < k; c++ {
			newCentroids[c] = make([]float64, dim)
		}

		for i := 0; i < n; i++ {
			c := assignments[i]
			counts[c]++
			for j := 0; j < dim; j++ {
				newCentroids[c][j] += matrix[i][j]
			}
		}

		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				cnt := float64(counts[c])
				var normSq float64
				for j := 0; j < dim; j++ {
					newCentroids[c][j] /= cnt
					normSq += newCentroids[c][j] * newCentroids[c][j]
				}
				if normSq > 0 {
					norm := math.Sqrt(normSq)
					for j := 0; j < dim; j++ {
						newCentroids[c][j] /= norm
					}
				}
				centroids[c] = newCentroids[c]
			}
		}
	}

	return assignments
}

// euclideanDistSq menghitung kuadrat jarak Euclidean antara dua vektor.
func euclideanDistSq(u, v []float64) float64 {
	var sum float64
	for i := 0; i < len(u); i++ {
		diff := u[i] - v[i]
		sum += diff * diff
	}
	return sum
}

// extractNGrams membersihkan dan mengekstrak unigram serta bigram dari teks.
func extractNGrams(text string) []string {
	words := strings.Fields(text)
	var filtered []string
	for _, w := range words {
		if len(w) >= 2 && !stopWordsID[w] {
			filtered = append(filtered, w)
		}
	}

	var ngrams []string
	// 1-Gram
	for _, w := range filtered {
		ngrams = append(ngrams, w)
	}
	// 2-Gram
	for i := 0; i < len(filtered)-1; i++ {
		ngrams = append(ngrams, filtered[i]+" "+filtered[i+1])
	}
	return ngrams
}

// cleanText menghapus karakter non-alfanumerik dan mengubah teks menjadi huruf kecil.
func cleanText(val string) string {
	val = strings.ToLower(strings.TrimSpace(val))
	var sb strings.Builder
	for _, r := range val {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// buildGroupName membuat label nama kelompok dari kata kunci utama.
func buildGroupName(keywords []string) string {
	if len(keywords) == 0 {
		return "Kelompok masalah umum"
	}

	limit := 3
	if len(keywords) < limit {
		limit = len(keywords)
	}

	titleWords := make([]string, limit)
	for i := 0; i < limit; i++ {
		titleWords[i] = titleCase(keywords[i])
	}

	return "Kelompok: " + strings.Join(titleWords, ", ")
}

// titleCase mengubah huruf awal setiap kata menjadi kapital.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// createFallbackGroup menghasilkan 1 kelompok fallback bila tidak ada kata kunci unik.
func (s *Service) createFallbackGroup(records []TextRecord, source string, requestedClusters int) *AnalysisResponse {
	var totalDamagedUnits int64
	items := make([]ClusterItem, len(records))

	for i, r := range records {
		if r.Source == "damage" {
			totalDamagedUnits += r.Quantity
		}
		items[i] = ClusterItem{
			ID:          r.ID,
			Source:      r.Source,
			Title:       r.Title,
			Description: r.Description,
			Quantity:    r.Quantity,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
		}
	}

	group := ClusterGroup{
		ClusterID:         1,
		Name:              "Kelompok masalah umum",
		Keywords:          []string{},
		ReportCount:       len(records),
		TotalDamagedUnits: totalDamagedUnits,
		Items:             items,
	}

	return &AnalysisResponse{
		Source:            source,
		TotalReports:      len(records),
		ClustersRequested: requestedClusters,
		Groups:            []ClusterGroup{group},
	}
}

// ParseClusters parsing parameter cluster ke integer.
func ParseClusters(value string) (int, error) {
	return strconv.Atoi(value)
}
