
"use strict";

(() => {
    // ==========================================
    // KONFIGURASI
    // ==========================================
    const isLocalDev = window.location.hostname === "localhost" && window.location.port === "8081";
    const API_BASE_URL = (((window.APP_CONFIG && window.APP_CONFIG.API_BASE_URL) || (isLocalDev ? "http://localhost:8080" : window.location.origin)) || "").replace(/\/+$/, "");
    const LOGIN_URL = "/login.html";

    // Sidebar berdasarkan role
    const SIDEBAR_URLS = {
        admin: "/client/components/sidebar.html",
        user: "/client/components/sidebar-user.html",
        teknisi: "/client/components/sidebar-teknisi.html"
    };

    // Dashboard berdasarkan role
    const ROLE_DASHBOARDS = {
        admin: "/client/admin/dashboard.html",
        user: "/client/user/dashboard.html",
        teknisi: "/client/teknisi/dashboard.html"
    };

    // ==========================================
    // CEGAH INISIALISASI BERULANG
    // ==========================================
    if (window.layoutInitialized) return;

    window.layoutInitialized = true;
    window.isRedirecting = false;
    window.currentUser = null;

    // ==========================================
    // REDIRECT KE LOGIN
    // ==========================================
    function redirectToLogin(
        message = "Sesi login berakhir. Silakan login kembali."
    ) {
        if (window.isRedirecting) return;

        window.isRedirecting = true;

        if (typeof Swal !== "undefined") {
            Swal.fire({
                icon: "warning",
                title: "Akses Tidak Valid",
                text: message,
                confirmButtonText: "Login Kembali",
                allowOutsideClick: false,
                allowEscapeKey: false
            }).then(() => {
                window.location.replace(LOGIN_URL);
            });
        } else {
            alert(message);
            window.location.replace(LOGIN_URL);
        }
    }

    window.redirectToLogin = redirectToLogin;

    // ==========================================
    // DETEKSI HALAMAN PUBLIK
    // ==========================================
    function isPublicPage() {
        const path = window.location.pathname.toLowerCase();

        return (
            path.endsWith("/login.html") ||
            path.endsWith("/register.html")
        );
    }

    // ==========================================
    // DETEKSI ROLE HALAMAN
    // ==========================================
    function getRequiredRole() {
        const path = window.location.pathname
            .replace(/\/+$/, "")
            .toLowerCase();

        const match = path.match(
            /\/(admin|teknisi|user)(?:\/|$)/
        );

        return match ? match[1] : null;
    }

    // ==========================================
    // CEK AUTENTIKASI DAN ROLE
    // ==========================================
    async function checkAuth() {
        if (isPublicPage()) return true;

        try {
            const response = await fetch(
                `${API_BASE_URL}/api/me`,
                {
                    method: "GET",
                    credentials: "include",
                    headers: {
                        Accept: "application/json"
                    },
                    cache: "no-store"
                }
            );

            if (response.status === 401) {
                redirectToLogin(
                    "Sesi login kedaluwarsa. Silakan login kembali."
                );
                return false;
            }

            if (response.status === 403) {
                redirectToLogin(
                    "Kamu tidak memiliki izin untuk mengakses halaman ini."
                );
                return false;
            }

            if (!response.ok) {
                console.error(
                    "[AUTH] Error API /api/me:",
                    response.status
                );

                Swal.fire({
                    icon: "error",
                    title: "Gagal Memeriksa Login",
                    text: "Tidak dapat memverifikasi akun. Silakan coba lagi."
                });

                return false;
            }

            const result = await response.json();
            const user = result.data || result.user;

            if (!user || !user.role) {
                redirectToLogin(
                    "Data pengguna atau role tidak valid."
                );
                return false;
            }

            // Normalisasi role
            const role = String(user.role)
                .trim()
                .toLowerCase();

            // Validasi role yang diperbolehkan
            if (
                !Object.prototype.hasOwnProperty.call(
                    ROLE_DASHBOARDS,
                    role
                )
            ) {
                console.error("[AUTH] Role tidak dikenal:", role);

                redirectToLogin(
                    "Role akun tidak dikenali. Hubungi administrator."
                );

                return false;
            }

            user.role = role;
            window.currentUser = user;

            console.log("[AUTH] Login valid. Role:", role);

            // ==========================================
            // PROTEKSI HALAMAN BERDASARKAN ROLE
            // ==========================================
            const requiredRole = getRequiredRole();

            if (requiredRole && requiredRole !== role) {
                console.warn(
                    `[AUTH] Role ${role} tidak boleh mengakses halaman ${requiredRole}.`
                );

                window.location.replace(
                    ROLE_DASHBOARDS[role]
                );

                return false;
            }

            // ==========================================
            // REDIRECT DASHBOARD UMUM
            // ==========================================
            if (!requiredRole) {
                const currentPath = window.location.pathname
                    .replace(/\/+$/, "")
                    .toLowerCase();

                const isGenericDashboard =
                    currentPath.endsWith("/dashboard.html") ||
                    currentPath === "/" ||
                    currentPath.endsWith("/index.html");

                if (isGenericDashboard) {
                    window.location.replace(
                        ROLE_DASHBOARDS[role]
                    );

                    return false;
                }
            }

            return true;

        } catch (error) {
            console.error(
                "[AUTH] Gagal memeriksa autentikasi:",
                error
            );

            Swal.fire({
                icon: "error",
                title: "Koneksi Bermasalah",
                text: "Server tidak dapat dihubungi. Periksa koneksi dan coba lagi."
            });

            return false;
        }
    }

    // ==========================================
    // PILIH SIDEBAR BERDASARKAN ROLE
    // ==========================================
    function getSidebarURL() {
        const role = window.currentUser?.role;

        if (!role) {
            console.error("[LAYOUT] Role pengguna tidak tersedia.");
            return null;
        }

        return SIDEBAR_URLS[role] || null;
    }

    // ==========================================
    // LOAD SIDEBAR
    // ==========================================
    async function loadSidebar() {
        const sidebar = document.getElementById(
            "sidebar-container"
        );

        if (!sidebar) {
            console.warn(
                "[LAYOUT] Sidebar container tidak ditemukan."
            );
            return;
        }

        const sidebarURL = getSidebarURL();

        if (!sidebarURL) {
            sidebar.innerHTML = `
                <div class="alert alert-danger m-3">
                    Role tidak valid. Sidebar tidak dapat dimuat.
                </div>
            `;
            return;
        }

        console.log("[LAYOUT] Memuat sidebar:", sidebarURL);

        try {
            const response = await fetch(sidebarURL, {
                cache: "no-cache"
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            sidebar.innerHTML = await response.text();

            // Atur menu aktif
            setActiveMenu(sidebar);

            // Aktifkan tombol logout
            setupLogout(sidebar);

            console.log(
                `[LAYOUT] Sidebar role ${window.currentUser.role} berhasil dimuat.`
            );

        } catch (error) {
            console.error(
                "[LAYOUT] Gagal memuat sidebar:",
                error
            );

            sidebar.innerHTML = `
                <div class="alert alert-danger m-3">
                    Sidebar gagal dimuat.
                    Periksa path file dan server frontend.
                </div>
            `;
        }
    }

    // ==========================================
    // MENU AKTIF
    // ==========================================
    function setActiveMenu(sidebar) {
        const currentPath = window.location.pathname
            .replace(/\/+$/, "")
            .toLowerCase();

        sidebar.querySelectorAll("a[href]").forEach(link => {
            const href = link.getAttribute("href");

            if (
                !href ||
                href.startsWith("#") ||
                href.startsWith("javascript:")
            ) {
                return;
            }

            try {
                const url = new URL(
                    href,
                    window.location.href
                );

                if (url.origin !== window.location.origin) {
                    return;
                }

                const linkPath = url.pathname
                    .replace(/\/+$/, "")
                    .toLowerCase();

                if (linkPath === currentPath) {
                    link.classList.add("active");
                    link.setAttribute("aria-current", "page");
                } else {
                    link.classList.remove("active");
                    link.removeAttribute("aria-current");
                }

            } catch (error) {
                console.warn(
                    "[LAYOUT] Link tidak valid:",
                    href
                );
            }
        });
    }

    // ==========================================
    // LOGOUT
    // ==========================================
    function setupLogout(sidebar) {
        const button = sidebar.querySelector("#logoutButton");

        if (!button) {
            console.warn(
                "[LAYOUT] Tombol logout tidak ditemukan."
            );
            return;
        }

        button.addEventListener("click", async event => {
            event.preventDefault();

            if (button.dataset.loading === "true") return;

            button.dataset.loading = "true";
            button.disabled = true;

            try {
                const response = await fetch(
                    `${API_BASE_URL}/api/auth/logout`,
                    {
                        method: "POST",
                        credentials: "include",
                        headers: {
                            Accept: "application/json"
                        }
                    }
                );

                if (!response.ok) {
                    console.error(
                        "[LAYOUT] Logout API gagal:",
                        response.status
                    );
                }

            } catch (error) {
                console.error(
                    "[LAYOUT] Request logout gagal:",
                    error
                );

            } finally {
                window.isRedirecting = true;
                window.location.replace(LOGIN_URL);
            }
        });
    }

    // ==========================================
    // INISIALISASI LAYOUT
    // ==========================================
    document.addEventListener(
        "DOMContentLoaded",
        async () => {
            console.log("[LAYOUT] Inisialisasi dimulai.");

            // Halaman login dan register tidak membutuhkan sidebar
            if (isPublicPage()) return;

            // Verifikasi login dan role terlebih dahulu
            const authenticated = await checkAuth();

            if (
                !authenticated ||
                window.isRedirecting
            ) {
                return;
            }

            // Sidebar dipilih setelah role terverifikasi
            await loadSidebar();
        }
    );
})();
