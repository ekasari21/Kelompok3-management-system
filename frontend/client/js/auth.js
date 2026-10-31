
"use strict";

(() => {
    // ==========================================
    // KONFIGURASI
    // ==========================================
    const isLocalDev = window.location.hostname === "localhost" && window.location.port === "8081";
    const API_BASE_URL = (((window.APP_CONFIG && window.APP_CONFIG.API_BASE_URL) || (isLocalDev ? "http://localhost:8080" : window.location.origin)) || "").replace(/\/+$/, "");
    const LOGIN_URL = "/login.html";

    // Hindari inisialisasi berulang
    if (window.authInitialized) {
        return;
    }

    window.authInitialized = true;
    window.isRedirecting = false;
    window.currentUser = null;

    // ==========================================
    // REDIRECT KE HALAMAN LOGIN
    // ==========================================
    window.redirectToLogin = function (
        message = "Sesi login berakhir. Silakan login kembali."
    ) {
        if (window.isRedirecting) {
            return;
        }

        window.isRedirecting = true;

        const redirect = () => {
            window.location.replace(LOGIN_URL);
        };

        if (typeof Swal !== "undefined") {
            Swal.fire({
                icon: "warning",
                title: "Sesi Berakhir!",
                text: message,
                confirmButtonText: "Login Kembali",
                allowOutsideClick: false,
                allowEscapeKey: false
            }).then(redirect);
        } else {
            Swal.fire({
                icon: "warning",
                title: "Sesi Berakhir!",
                text: message,
                confirmButtonText: "Login Kembali",
                allowOutsideClick: false,
                allowEscapeKey: false
            }).then(redirect);
        }
    };

// ==========================================
// PEMERIKSAAN AUTENTIKASI
// ==========================================
window.authReady = (async () => {
    const pathname = window.location.pathname
        .replace(/\/+$/, "")
        .toLowerCase();

    // Tidak memeriksa sesi pada halaman login/register
    if (
        pathname.endsWith("/login.html") ||
        pathname.endsWith("/register.html")
    ) {
        return true;
    }

    // Mengambil message dari respons backend
    async function getResponseMessage(response) {
        try {
            const result = await response.clone().json();
            return result.message || result.error || null;
        } catch {
            return null;
        }
    }

    // Menampilkan pesan SweetAlert2
    async function showAuthMessage(message, icon = "warning") {
        if (typeof Swal !== "undefined") {
            await Swal.fire({
                icon: icon,
                title: icon === "error"
                    ? "Terjadi Kesalahan"
                    : "Informasi",
                text: message,
                confirmButtonText: "OK",
                allowOutsideClick: false
            });
        } else {
            alert(message);
        }
    }

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

        // Ambil message dari backend jika tersedia
        const message = await getResponseMessage(response);

        // ==========================================
        // SESI TIDAK VALID / KEDALUWARSA
        // ==========================================
        if (response.status === 401) {
            await showAuthMessage(
                message || "Sesi Anda telah berakhir. Silakan login kembali.",
                "warning"
            );

            window.redirectToLogin();
            return false;
        }

        // ==========================================
        // AKSES DITOLAK
        // ==========================================
        if (response.status === 403) {
            await showAuthMessage(
                message || "Akses ditolak. Anda tidak memiliki izin.",
                "error"
            );

            window.redirectToLogin();
            return false;
        }

        // ==========================================
        // ERROR SERVER
        // ==========================================
        if (!response.ok) {
            console.error(
                "[AUTH] Gagal memeriksa sesi:",
                response.status,
                response.statusText
            );

            await showAuthMessage(
                message || "Gagal memeriksa sesi. Silakan coba kembali.",
                "error"
            );

            return false;
        }

        // ==========================================
        // MEMBACA DATA PENGGUNA
        // ==========================================
        const result = await response.json();
        const user = result.data || result.user;

        if (!user || !user.role) {
            console.error(
                "[AUTH] Data pengguna atau role tidak ditemukan."
            );

            await showAuthMessage(
                result.message ||
                "Data sesi tidak valid. Silakan login kembali.",
                "warning"
            );

            window.redirectToLogin();
            return false;
        }

        // ==========================================
        // SIMPAN INFORMASI PENGGUNA
        // ==========================================
        window.currentUser = user;

        console.log(
            "[AUTH] Autentikasi berhasil:",
            user.role
        );

        // Jika respons sukses juga mengandung message,
        // tampilkan pesan tersebut.
        if (result.message) {
            await showAuthMessage(result.message, "success");
        }

        return true;

    } catch (error) {
        console.error(
            "[AUTH] Tidak dapat memeriksa sesi:",
            error
        );

        await showAuthMessage(
            "Tidak dapat terhubung ke server. Periksa koneksi internet atau server backend.",
            "error"
        );

        // Gangguan jaringan tidak langsung dianggap
        // sebagai sesi kedaluwarsa.
        return false;
    }
})();

})();
