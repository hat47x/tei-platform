package xteaControl;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Loopback JSON control surface for X-TEA Swing processes.
 *
 * The server is deliberately small and dependency-free. It is not a general
 * RPC surface: only the fixed routes listed in {@link #ROUTES} are accepted.
 */
final class BridgeServer {
    static final List<String> ROUTES = new ArrayList<String>();

    static {
        ROUTES.add("GET /health");
        ROUTES.add("GET /windows");
        ROUTES.add("POST /snapshot");
        ROUTES.add("POST /menus");
        ROUTES.add("POST /invoke");
        ROUTES.add("POST /set");
        ROUTES.add("POST /select");
        ROUTES.add("POST /table/set");
        ROUTES.add("POST /table/read");
        ROUTES.add("POST /key");
        ROUTES.add("POST /close");
        ROUTES.add("POST /wait");
        ROUTES.add("POST /screenshot");
        ROUTES.add("GET /trace");
        ROUTES.add("GET /routes");
        ROUTES.add("POST /shutdown");
    }

    private final HttpServer server;
    private final String token;
    private final String endpoint;
    private final AtomicBoolean shuttingDown = new AtomicBoolean(false);
    private final String appName;

    BridgeServer(String bind, int port, String token, boolean allowRemote) throws IOException {
        InetAddress address = InetAddress.getByName(bind);
        if (!allowRemote && !address.isLoopbackAddress()) {
            throw new IllegalArgumentException("refusing non-loopback bind without allowRemote=true");
        }
        this.token = token;
        this.server = HttpServer.create(new InetSocketAddress(address, port), 0);
        this.appName = System.getProperty("sun.java.command", "unknown");
        String host = address.getHostAddress();
        if (host.indexOf(':') >= 0) {
            host = "[" + host + "]";
        }
        this.endpoint = "http://" + host + ":" + server.getAddress().getPort();
        server.createContext("/", new Handler());
        server.setExecutor(Executors.newFixedThreadPool(4, new DaemonFactory()));
    }

    String endpoint() {
        return endpoint;
    }

    void start() {
        server.start();
    }

    void stop() {
        if (shuttingDown.compareAndSet(false, true)) {
            server.stop(0);
        }
    }

    private final class Handler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            long started = System.currentTimeMillis();
            try {
                String route = exchange.getRequestMethod() + " " + exchange.getRequestURI().getPath();
                if (!authorized(exchange)) {
                    respond(exchange, 401, failure("UNAUTHORIZED", "missing or invalid token"));
                    return;
                }
                Map<String, Object> body = readBody(exchange);
                if ("GET /health".equals(route)) {
                    respond(exchange, 200, ok(health()));
                } else if ("GET /windows".equals(route)) {
                    respond(exchange, 200, ok(Ui.windows()));
                } else if ("POST /snapshot".equals(route)) {
                    respond(exchange, 200, ok(Ui.snapshot(body)));
                } else if ("POST /menus".equals(route)) {
                    respond(exchange, 200, ok(Ui.menuCatalog()));
                } else if ("POST /invoke".equals(route)) {
                    respond(exchange, 200, ok(Ui.invoke(selector(body), longValue(body.get("settleMs"), 0))));
                } else if ("POST /set".equals(route)) {
                    respond(exchange, 200, ok(Ui.setValue(selector(body), body.get("value"),
                            stringValue(body.get("mode")))));
                } else if ("POST /select".equals(route)) {
                    respond(exchange, 200, ok(Ui.select(selector(body), body)));
                } else if ("POST /table/set".equals(route)) {
                    respond(exchange, 200, ok(Ui.tableSet(selector(body), body)));
                } else if ("POST /table/read".equals(route)) {
                    respond(exchange, 200, ok(Ui.tableRead(selector(body), body)));
                } else if ("POST /key".equals(route)) {
                    respond(exchange, 200, ok(Ui.key(body.get("selector") instanceof Map
                            ? selector(body) : null, body)));
                } else if ("POST /close".equals(route)) {
                    respond(exchange, 200, ok(Ui.close(selector(body))));
                } else if ("POST /wait".equals(route)) {
                    respond(exchange, 200, ok(Ui.waitFor(body)));
                } else if ("POST /screenshot".equals(route)) {
                    respond(exchange, 200, ok(Ui.screenshot(selector(body))));
                } else if ("GET /trace".equals(route)) {
                    respond(exchange, 200, ok(Trace.since(longValue(query(exchange, "since"), 0))));
                } else if ("GET /routes".equals(route)) {
                    Map<String, Object> routes = new LinkedHashMap<String, Object>();
                    routes.put("routes", ROUTES);
                    respond(exchange, 200, ok(routes));
                } else if ("POST /shutdown".equals(route)) {
                    Map<String, Object> value = new LinkedHashMap<String, Object>();
                    value.put("stopping", Boolean.TRUE);
                    respond(exchange, 200, ok(value));
                    final boolean exit = boolValue(body.get("exit"), false);
                    Thread stopper = new Thread(new Runnable() {
                        @Override
                        public void run() {
                            try {
                                Thread.sleep(100L);
                            } catch (InterruptedException e) {
                                Thread.currentThread().interrupt();
                            }
                            stop();
                            if (exit) {
                                System.exit(0);
                            }
                        }
                    }, "xtea-control-shutdown");
                    stopper.setDaemon(true);
                    stopper.start();
                } else {
                    respond(exchange, 404, failure("NOT_FOUND", "unknown route: " + route));
                }
            } catch (Ui.BridgeException e) {
                respond(exchange, statusFor(e.code), failure(e.code, e.getMessage()));
            } catch (IllegalArgumentException e) {
                respond(exchange, 400, failure("INVALID_ARGUMENT", e.getMessage()));
            } catch (Throwable t) {
                respond(exchange, 500, failure("INTERNAL_ERROR", describe(t)));
            } finally {
                exchange.close();
            }
            Trace.record("http", exchange.getRequestMethod() + " " + exchange.getRequestURI().getPath(),
                    "durationMs=" + (System.currentTimeMillis() - started));
        }
    }

    private Map<String, Object> health() {
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("ok", Boolean.TRUE);
        value.put("protocol", "xtea-control/1");
        value.put("pid", Long.valueOf(pid()));
        value.put("app", appName);
        value.put("java", System.getProperty("java.version"));
        value.put("startedAt", Agent.STARTED_AT);
        value.put("os", System.getProperty("os.name") + " " + System.getProperty("os.arch"));
        value.put("windows", Ui.windowTitles());
        return value;
    }

    private boolean authorized(HttpExchange exchange) {
        if (token == null || token.isEmpty()) {
            return true;
        }
        String provided = exchange.getRequestHeaders().getFirst("X-Xtea-Token");
        if (provided == null) {
            String authorization = exchange.getRequestHeaders().getFirst("Authorization");
            if (authorization != null && authorization.startsWith("Bearer ")) {
                provided = authorization.substring("Bearer ".length()).trim();
            }
        }
        return constantTimeEquals(token, provided);
    }

    private static boolean constantTimeEquals(String expected, String actual) {
        if (actual == null) {
            return false;
        }
        byte[] left = expected.getBytes(StandardCharsets.UTF_8);
        byte[] right = actual.getBytes(StandardCharsets.UTF_8);
        int difference = left.length ^ right.length;
        for (int i = 0; i < Math.max(left.length, right.length); i++) {
            byte l = i < left.length ? left[i] : 0;
            byte r = i < right.length ? right[i] : 0;
            difference |= l ^ r;
        }
        return difference == 0;
    }

    private Map<String, Object> readBody(HttpExchange exchange) throws IOException {
        if (!"POST".equalsIgnoreCase(exchange.getRequestMethod())) {
            return new LinkedHashMap<String, Object>();
        }
        InputStream input = exchange.getRequestBody();
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        byte[] buffer = new byte[8192];
        int total = 0;
        int read;
        while ((read = input.read(buffer)) >= 0) {
            total += read;
            if (total > 2 * 1024 * 1024) {
                throw new IllegalArgumentException("request body too large");
            }
            bytes.write(buffer, 0, read);
        }
        String text = new String(bytes.toByteArray(), StandardCharsets.UTF_8).trim();
        if (text.isEmpty()) {
            return new LinkedHashMap<String, Object>();
        }
        return Json.parseObject(text);
    }

    private void respond(HttpExchange exchange, int status, Map<String, Object> body) throws IOException {
        byte[] bytes = Json.write(body).getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json; charset=utf-8");
        exchange.getResponseHeaders().set("Cache-Control", "no-store");
        exchange.sendResponseHeaders(status, bytes.length);
        OutputStream output = exchange.getResponseBody();
        output.write(bytes);
        output.flush();
    }

    private static Map<String, Object> ok(Object value) {
        Map<String, Object> response = new LinkedHashMap<String, Object>();
        response.put("ok", Boolean.TRUE);
        response.put("value", value);
        return response;
    }

    private static Map<String, Object> failure(String code, String message) {
        Map<String, Object> error = new LinkedHashMap<String, Object>();
        error.put("code", code);
        error.put("message", message);
        Map<String, Object> response = new LinkedHashMap<String, Object>();
        response.put("ok", Boolean.FALSE);
        response.put("error", error);
        return response;
    }

    private static int statusFor(String code) {
        if ("NOT_FOUND".equals(code) || "PATH_NOT_FOUND".equals(code) || "ITEM_NOT_FOUND".equals(code)) {
            return 404;
        }
        if ("EDT_TIMEOUT".equals(code)) {
            return 504;
        }
        if (code.startsWith("INVALID_") || "NOT_A_TABLE".equals(code) || "CELL_NOT_EDITABLE".equals(code)
                || "UNSUPPORTED_COMPONENT".equals(code) || "NO_WINDOW".equals(code)) {
            return 400;
        }
        return 500;
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> selector(Map<String, Object> body) {
        Object selector = body.get("selector");
        if (selector instanceof Map) {
            return (Map<String, Object>) selector;
        }
        // Convenience: accept selector keys at the top level as well.
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        for (String key : new String[] { "window", "path", "role", "text", "textContains", "name", "class", "index" }) {
            if (body.containsKey(key)) {
                value.put(key, body.get(key));
            }
        }
        return value;
    }

    private static String query(HttpExchange exchange, String key) {
        String query = exchange.getRequestURI().getQuery();
        if (query == null) {
            return null;
        }
        for (String pair : query.split("&")) {
            int separator = pair.indexOf('=');
            if (separator > 0 && pair.substring(0, separator).equals(key)) {
                return pair.substring(separator + 1);
            }
        }
        return null;
    }

    private static String stringValue(Object value) {
        return value == null ? null : String.valueOf(value);
    }

    private static long longValue(Object value, long fallback) {
        if (value instanceof Number) {
            return ((Number) value).longValue();
        }
        if (value != null) {
            try {
                return Long.parseLong(String.valueOf(value));
            } catch (NumberFormatException e) {
                return fallback;
            }
        }
        return fallback;
    }

    private static boolean boolValue(Object value, boolean fallback) {
        if (value instanceof Boolean) {
            return ((Boolean) value).booleanValue();
        }
        return value == null ? fallback : Boolean.parseBoolean(String.valueOf(value));
    }

    private static String describe(Throwable throwable) {
        String message = throwable.getMessage();
        return throwable.getClass().getSimpleName() + (message == null ? "" : ": " + message);
    }

    private static long pid() {
        String name = java.lang.management.ManagementFactory.getRuntimeMXBean().getName();
        int separator = name.indexOf('@');
        if (separator > 0) {
            try {
                return Long.parseLong(name.substring(0, separator));
            } catch (NumberFormatException e) {
                // fall through
            }
        }
        return -1L;
    }

    private static final class DaemonFactory implements ThreadFactory {
        @Override
        public Thread newThread(Runnable runnable) {
            Thread thread = new Thread(runnable, "xtea-control-http");
            thread.setDaemon(true);
            return thread;
        }
    }
}
