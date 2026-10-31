/**
 * Dashboard Teknisi - Inventaris Cerdas
 * Pusat monitoring antrean keluhan dan status pemeliharaan fasilitas
 */

"use strict";

$(document).ready(function () {
    const isLocalDev = window.location.hostname === "localhost" && window.location.port === "8081";
    const API_BASE = (window.APP_CONFIG && window.APP_CONFIG.API_BASE_URL) || window.API_BASE || (isLocalDev ? "http://localhost:8080" : window.location.origin);

    const API_ME = API_BASE + "/api/me";
    const API_COMPLAINTS = API_BASE + "/api/private/complaints";

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
        if (Number.isNaN(date.getTime())) return escapeHtml(value);
        return date.toLocaleDateString("id-ID", {
            day: "2-digit",
            month: "short",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit"
        });
    }

    function getPriorityBadge(priority) {
        const p = Number(priority || 0);
        if (p >= 3) {
            return `<span class="badge bg-danger-subtle text-danger border border-danger-subtle px-2 py-1"><i class="bi bi-exclamation-octagon-fill me-1"></i>Kritis</span>`;
        } else if (p >= 2) {
            return `<span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle px-2 py-1"><i class="bi bi-exclamation-triangle-fill me-1"></i>Tinggi</span>`;
        } else if (p >= 1) {
            return `<span class="badge bg-primary-subtle text-primary border border-primary-subtle px-2 py-1">Sedang</span>`;
        }
        return `<span class="badge bg-info-subtle text-info-emphasis border border-info-subtle px-2 py-1">Rendah</span>`;
    }

    function getStatusBadge(status, isFixed) {
        const norm = String(status || "pending").toLowerCase();
        if (isFixed || norm === "resolved" || norm === "selesai") {
            return `<span class="badge bg-success-subtle text-success border border-success-subtle px-2 py-1"><i class="bi bi-check-circle me-1"></i>Selesai Diperbaiki</span>`;
        }
        if (norm === "unrepairable") {
            return `<span class="badge bg-danger text-white px-2 py-1"><i class="bi bi-x-octagon me-1"></i>Rusak Tidak Bisa Diperbaiki</span>`;
        }
        if (norm === "in_progress" || norm === "diproses") {
            return `<span class="badge bg-info-subtle text-info-emphasis border border-info-subtle px-2 py-1"><i class="bi bi-gear me-1"></i>Sedang Ditangani</span>`;
        }
        return `<span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle px-2 py-1"><i class="bi bi-clock me-1"></i>Menunggu Tindakan</span>`;
    }

    // 1. Profil Teknisi
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
                const name = currentUser.name || currentUser.username || "Teknisi";
                const email = currentUser.email || "-";

                $("#topbarTeknisiName, #welcomeTeknisiName").text(name);
                $("#topbarTeknisiEmail").text(email);
                $("#topbarTeknisiRole").text((currentUser.role || "teknisi").toUpperCase());
            }
        } catch (err) {
            console.warn("[DASHBOARD TEKNISI] Gagal memuat profil:", err);
        }
    }

    // 2. Data Keluhan Aset
    async function loadComplaintsDashboard() {
        try {
            const res = await fetch(API_COMPLAINTS, {
                method: "GET",
                credentials: "include",
                headers: { "Accept": "application/json" }
            });

            let complaints = [];
            if (res.ok) {
                const json = await res.json();
                if (Array.isArray(json)) {
                    complaints = json;
                } else if (Array.isArray(json.data)) {
                    complaints = json.data;
                }
            }

            // Hitung statistik
            const total = complaints.length;
            let pendingCount = 0;
            let inProgressCount = 0;
            let resolvedCount = 0;
            let unrepairableCount = 0;

            complaints.forEach(c => {
                const st = String(c.status || "").toLowerCase();
                if (c.is_fixed || st === "resolved" || st === "selesai") {
                    resolvedCount++;
                } else if (st === "unrepairable") {
                    unrepairableCount++;
                } else if (st === "in_progress" || st === "diproses") {
                    inProgressCount++;
                } else {
                    pendingCount++;
                }
            });

            $("#statTotal").text(total);
            $("#statPending").text(pendingCount);
            $("#statInProgress").text(inProgressCount);
            $("#statResolved").text(resolvedCount);
            $("#statUnrepairable").text(unrepairableCount);

            // Perhitungan progress bar
            if (total > 0) {
                const pctResolved = Math.round((resolvedCount / total) * 100);
                const pctInProgress = Math.round((inProgressCount / total) * 100);
                const pctPending = Math.round((pendingCount / total) * 100);
                const pctUnrepairable = Math.round((unrepairableCount / total) * 100);

                $("#barResolved").css("width", pctResolved + "%");
                $("#barInProgress").css("width", pctInProgress + "%");
                $("#barPending").css("width", pctPending + "%");
                $("#barUnrepairable").css("width", pctUnrepairable + "%");

                $("#pctResolvedText").text(resolvedCount + ` (${pctResolved}%)`);
                $("#pctInProgressText").text(inProgressCount + ` (${pctInProgress}%)`);
                $("#pctPendingText").text(pendingCount + ` (${pctPending}%)`);
                $("#pctUnrepairableText").text(unrepairableCount + ` (${pctUnrepairable}%)`);
            }

            // Render antrean keluhan (maksimal 6 keluhan yang butuh perhatian)
            const $tbody = $("#tableComplaintQueueBody");
            $tbody.empty();

            if (complaints.length === 0) {
                $tbody.html(`
                    <tr>
                        <td colspan="7" class="text-center py-4 text-muted">
                            <i class="bi bi-check2-circle fs-3 d-block mb-2 text-success opacity-70"></i>
                            Tidak ada keluhan fasilitas yang perlu ditangani saat ini.<br>
                            Semua fasilitas dalam kondisi baik.
                        </td>
                    </tr>
                `);
                return;
            }

            // Prioritaskan yang belum selesai (pending, in_progress, dll.)
            const activeComplaints = complaints.slice().sort((a, b) => {
                const isFixedA = a.is_fixed || a.status === "resolved" || a.status === "unrepairable";
                const isFixedB = b.is_fixed || b.status === "resolved" || b.status === "unrepairable";
                if (isFixedA !== isFixedB) return isFixedA ? 1 : -1;
                return new Date(b.created_at || 0) - new Date(a.created_at || 0);
            }).slice(0, 6);

            activeComplaints.forEach(c => {
                const code = c.code || "-";
                const date = formatDate(c.created_at);
                const reporter = c.user?.name || c.user?.email || "Pengguna";
                const assetName = c.asset?.nama_asset || c.assetName || "Aset Inventaris";
                const title = c.title || "-";
                const priorityBadge = getPriorityBadge(c.priority);
                const statusBadge = getStatusBadge(c.status, c.is_fixed);

                $tbody.append(`
                    <tr>
                        <td>
                            <span class="badge bg-light text-primary border fw-semibold">${escapeHtml(code)}</span>
                            <div class="text-muted small mt-1">${date}</div>
                        </td>
                        <td>
                            <div class="fw-bold text-dark">${escapeHtml(reporter)}</div>
                        </td>
                        <td>
                            <div class="fw-semibold text-dark">${escapeHtml(assetName)}</div>
                        </td>
                        <td>
                            <div class="text-truncate" style="max-width: 200px;" title="${escapeHtml(title)}">${escapeHtml(title)}</div>
                        </td>
                        <td>${priorityBadge}</td>
                        <td>${statusBadge}</td>
                        <td class="text-end">
                            <a href="complaints.html" class="btn btn-sm btn-primary" title="Tindak Lanjuti di Halaman Keluhan">
                                <i class="bi bi-tools me-1"></i> Tindak Lanjut
                            </a>
                        </td>
                    </tr>
                `);
            });

        } catch (err) {
            console.error("[DASHBOARD TEKNISI] Gagal memuat keluhan:", err);
            $("#tableComplaintQueueBody").html(`
                <tr>
                    <td colspan="7" class="text-center text-danger py-3">
                        Gagal memuat antrean keluhan teknisi.
                    </td>
                </tr>
            `);
        }
    }

    // Inisialisasi
    loadUserProfile();
    loadComplaintsDashboard();
});
