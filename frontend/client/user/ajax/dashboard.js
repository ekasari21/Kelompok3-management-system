/**
 * Dashboard User (Self Pattern) - Inventaris Cerdas
 * Menampilkan ringkasan data peminjaman dan keluhan mandiri (milik pengguna itu sendiri)
 */

"use strict";

$(document).ready(function () {
    const isLocalDev = window.location.hostname === "localhost" && window.location.port === "8081";
    const API_BASE = (window.APP_CONFIG && window.APP_CONFIG.API_BASE_URL) || window.API_BASE || (isLocalDev ? "http://localhost:8080" : window.location.origin);

    const API_ME = API_BASE + "/api/me";
    const API_MY_LENDING = API_BASE + "/api/private/asset-lending";
    const API_MY_COMPLAINTS = API_BASE + "/api/private/complaints";

    let currentUser = null;

    function escapeHtml(value) {
        return String(value ?? "").replace(/[&<>"']/g, function (char) {
            const map = {
                "&": "&amp;",
                "<": "&lt;",
                ">": "&gt;",
                '"': "&quot;",
                "'": "&#039;"
            };
            return map[char];
        });
    }

    function formatDate(value) {
        if (!value) return "-";
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            // Coba potong YYYY-MM-DD
            const str = String(value).substring(0, 10);
            const parts = str.split("-");
            if (parts.length === 3) return `${parts[2]}/${parts[1]}/${parts[0]}`;
            return escapeHtml(value);
        }
        return date.toLocaleDateString("id-ID", {
            day: "2-digit",
            month: "short",
            year: "numeric"
        });
    }

    function getLendingBadge(status) {
        const norm = String(status || "menunggu").toLowerCase().replace(/\s+/g, "_");
        switch (norm) {
            case "dipinjam":
            case "borrowed":
                return `<span class="badge bg-success-subtle text-success border border-success-subtle px-2 py-1"><i class="bi bi-arrow-repeat me-1"></i>Sedang Dipinjam</span>`;
            case "menunggu":
            case "pending":
                return `<span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle px-2 py-1"><i class="bi bi-clock me-1"></i>Menunggu Persetujuan</span>`;
            case "selesai":
            case "dikembalikan":
            case "returned":
                return `<span class="badge bg-primary-subtle text-primary border border-primary-subtle px-2 py-1"><i class="bi bi-check2-circle me-1"></i>Selesai</span>`;
            case "ditolak":
            case "rejected":
                return `<span class="badge bg-danger-subtle text-danger border border-danger-subtle px-2 py-1"><i class="bi bi-x-circle me-1"></i>Ditolak</span>`;
            default:
                return `<span class="badge bg-secondary-subtle text-secondary px-2 py-1">${escapeHtml(status)}</span>`;
        }
    }

    function getComplaintBadge(status, isFixed) {
        const norm = String(status || "low").toLowerCase();
        if (isFixed || norm === "resolved" || norm === "selesai") {
            return `<span class="badge bg-success-subtle text-success border border-success-subtle px-2 py-1"><i class="bi bi-check-circle me-1"></i>Selesai Diperbaiki</span>`;
        }
        if (norm === "unrepairable") {
            return `<span class="badge bg-danger text-white px-2 py-1"><i class="bi bi-x-octagon me-1"></i>Rusak Tidak Bisa Diperbaiki</span>`;
        }
        if (norm === "in_progress" || norm === "diproses") {
            return `<span class="badge bg-info-subtle text-info-emphasis border border-info-subtle px-2 py-1"><i class="bi bi-gear me-1"></i>Sedang Dikerjakan</span>`;
        }
        return `<span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle px-2 py-1"><i class="bi bi-hourglass me-1"></i>Menunggu Respon</span>`;
    }

    function getPriorityBadge(priority) {
        const p = Number(priority || 0);
        if (p >= 3) {
            return `<span class="badge bg-danger-subtle text-danger border border-danger-subtle">Kritis</span>`;
        } else if (p >= 2) {
            return `<span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle">Tinggi</span>`;
        } else if (p >= 1) {
            return `<span class="badge bg-primary-subtle text-primary border border-primary-subtle">Sedang</span>`;
        }
        return `<span class="badge bg-info-subtle text-info-emphasis border border-info-subtle">Rendah</span>`;
    }

    // 1. Ambil Profil User
    async function loadUserProfile() {
        try {
            const res = await fetch(API_ME, {
                method: "GET",
                credentials: "include",
                headers: { "Accept": "application/json" }
            });

            if (res.ok) {
                const json = await res.json();
                currentUser = json.data || json.user || json;
                const name = currentUser.name || currentUser.username || "Pengguna";
                const email = currentUser.email || "-";

                $("#topbarUserName, #welcomeUserName").text(name);
                $("#topbarUserEmail").text(email);
                $("#topbarUserRole").text((currentUser.role || "user").toUpperCase());
            }
        } catch (err) {
            console.warn("[DASHBOARD] Gagal memuat profil user:", err);
        }
    }

    // 2. Ambil Riwayat Peminjaman Mandiri (Self Lending)
    async function loadMyLendingData() {
        try {
            const res = await fetch(API_MY_LENDING, {
                method: "GET",
                credentials: "include",
                headers: { "Accept": "application/json" }
            });

            let items = [];
            if (res.ok) {
                const json = await res.json();
                if (Array.isArray(json)) {
                    items = json;
                } else if (Array.isArray(json.data)) {
                    items = json.data;
                }
            }

            // Hitung statistik
            const total = items.length;
            let activeCount = 0;
            let pendingCount = 0;
            let returnedCount = 0;

            items.forEach(item => {
                const st = String(item.lending_status || item.status || "").toLowerCase();
                if (st === "dipinjam" || st === "borrowed") {
                    activeCount += (item.qty_lending || 1);
                } else if (st === "menunggu" || st === "pending") {
                    pendingCount += (item.qty_lending || 1);
                } else if (st === "selesai" || st === "dikembalikan" || st === "returned") {
                    returnedCount += (item.qty_lending || 1);
                }
            });

            $("#statMyTotalLending").text(total);
            $("#statMyActiveLending").text(activeCount);
            $("#statMyPendingLending").text(pendingCount);

            // Render 5 peminjaman terkini
            const $tbody = $("#tableMyLendingBody");
            $tbody.empty();

            if (items.length === 0) {
                $tbody.html(`
                    <tr>
                        <td colspan="5" class="text-center py-4 text-muted">
                            <i class="bi bi-box-arrow-up-right fs-3 d-block mb-2 text-secondary opacity-50"></i>
                            Belum ada riwayat peminjaman aset mandiri.<br>
                            <a href="asset-lending.html" class="btn btn-sm btn-primary mt-2">
                                <i class="bi bi-plus-lg me-1"></i> Ajukan Pinjam Aset Sekarang
                            </a>
                        </td>
                    </tr>
                `);
                return;
            }

            // Urutkan terbaru di atas
            const recent = items.slice().reverse().slice(0, 5);

            recent.forEach(item => {
                const code = item.code || item.kode || `PJM-${item.lending_id || item.id}`;
                const assetName = item.asset_detail?.nama_asset || item.asset?.nama_asset || item.assetName || item.nama_asset || "Aset Inventaris";
                const borrowDate = formatDate(item.lending_date || item.tanggalPinjam || item.borrowDate);
                const returnDate = formatDate(item.lending_return || item.tanggalKembali || item.returnDate);
                const status = item.lending_status || item.status;

                $tbody.append(`
                    <tr>
                        <td><span class="badge bg-light text-primary border fw-semibold">${escapeHtml(code)}</span></td>
                        <td>
                            <div class="fw-bold text-dark">${escapeHtml(assetName)}</div>
                            <small class="text-muted">Jumlah: ${item.qty_lending || 1} unit</small>
                        </td>
                        <td>
                            <div><i class="bi bi-calendar-event me-1 text-primary"></i> ${borrowDate}</div>
                            <small class="text-muted"><i class="bi bi-calendar-check me-1"></i> ${returnDate}</small>
                        </td>
                        <td>${getLendingBadge(status)}</td>
                        <td class="text-end">
                            <a href="asset-lending.html" class="btn btn-sm btn-outline-primary" title="Lihat Detail di Halaman Peminjaman">
                                <i class="bi bi-eye"></i>
                            </a>
                        </td>
                    </tr>
                `);
            });

        } catch (err) {
            console.error("[DASHBOARD] Gagal memuat data peminjaman mandiri:", err);
            $("#tableMyLendingBody").html(`
                <tr>
                    <td colspan="5" class="text-center text-danger py-3">
                        Gagal memuat data peminjaman mandiri.
                    </td>
                </tr>
            `);
        }
    }

    // 3. Ambil Riwayat Keluhan Mandiri (Self Complaints)
    async function loadMyComplaintsData() {
        try {
            const res = await fetch(API_MY_COMPLAINTS, {
                method: "GET",
                credentials: "include",
                headers: { "Accept": "application/json" }
            });

            let items = [];
            if (res.ok) {
                const json = await res.json();
                if (Array.isArray(json)) {
                    items = json;
                } else if (Array.isArray(json.data)) {
                    items = json.data;
                }
            }

            // Hitung statistik
            const total = items.length;
            $("#statMyTotalComplaints").text(total);

            // Render 5 keluhan terkini
            const $tbody = $("#tableMyComplaintsBody");
            $tbody.empty();

            if (items.length === 0) {
                $tbody.html(`
                    <tr>
                        <td colspan="5" class="text-center py-4 text-muted">
                            <i class="bi bi-chat-left-text fs-3 d-block mb-2 text-secondary opacity-50"></i>
                            Belum ada keluhan fasilitas yang dilaporkan.<br>
                            <a href="complaints.html" class="btn btn-sm btn-outline-danger mt-2">
                                <i class="bi bi-plus-lg me-1"></i> Buat Laporan Keluhan Baru
                            </a>
                        </td>
                    </tr>
                `);
                return;
            }

            // Urutkan keluhan terbaru
            const recent = items.slice().reverse().slice(0, 5);

            recent.forEach(c => {
                const code = c.code || "-";
                const assetName = c.asset?.nama_asset || c.assetName || "Fasilitas Kantor";
                const title = c.title || "-";
                const date = formatDate(c.created_at);
                const statusBadge = getComplaintBadge(c.status, c.is_fixed);
                const priorityBadge = getPriorityBadge(c.priority);

                $tbody.append(`
                    <tr>
                        <td>
                            <div class="fw-semibold text-primary">${escapeHtml(code)}</div>
                            <small class="text-muted">${date}</small>
                        </td>
                        <td>
                            <div class="fw-bold text-dark">${escapeHtml(assetName)}</div>
                            <small class="text-muted text-truncate d-inline-block" style="max-width: 180px;">${escapeHtml(title)}</small>
                        </td>
                        <td>${priorityBadge}</td>
                        <td>${statusBadge}</td>
                        <td class="text-end">
                            <a href="complaints.html" class="btn btn-sm btn-outline-secondary" title="Lihat Detail di Halaman Keluhan">
                                <i class="bi bi-arrow-right"></i>
                            </a>
                        </td>
                    </tr>
                `);
            });

        } catch (err) {
            console.error("[DASHBOARD] Gagal memuat data keluhan mandiri:", err);
            $("#tableMyComplaintsBody").html(`
                <tr>
                    <td colspan="5" class="text-center text-danger py-3">
                        Gagal memuat data keluhan mandiri.
                    </td>
                </tr>
            `);
        }
    }

    // Inisialisasi
    loadUserProfile();
    loadMyLendingData();
    loadMyComplaintsData();
});
