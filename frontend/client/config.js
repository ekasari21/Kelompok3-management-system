(function () {
    const hostname = window.location.hostname;
    const port = window.location.port;

    const isLocalDev =
        hostname === "localhost" && port === "8081";

    const isProduction =
        !isLocalDev;

    const API_URL = isLocalDev
        ? "http://localhost:8080"
        : "https://smart-stock.hostless.app/";

    window.APP_CONFIG = {
        API_BASE_URL: API_URL,
        BASIC_URL: window.location.origin,
    };

})();
