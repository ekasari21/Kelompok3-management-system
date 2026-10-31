(function () {
    const hostname = window.location.hostname;
    const port = window.location.port;

    const isLocalDev =
        hostname === "localhost" && port === "8081";

    // Arahkan ke backend port 8080 jika dijalankan lokal dev di port 8081.
    // Jika di hosting produksi:
    // Gunakan window.location.origin secara dinamis agar sesuai domain hosting saat ini.
    // Jika diakses dari domain smart-stock.hostless.app, gunakan domain tersebut.
    let apiUrl = window.location.origin;
    if (isLocalDev) {
        apiUrl = "http://localhost:8080";
    } else if (hostname.includes("smart-stock.hostless.app")) {
        apiUrl = "https://smart-stock.hostless.app";
    }

    // Pastikan tidak ada trailing slash di akhir URL untuk mencegah error double-slash (//api)
    apiUrl = (apiUrl || "").replace(/\/+$/, "");

    window.APP_CONFIG = {
        API_BASE_URL: apiUrl,
        BASIC_URL: window.location.origin.replace(/\/+$/, ""),
    };
})();
