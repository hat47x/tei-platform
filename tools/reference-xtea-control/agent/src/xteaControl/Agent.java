package xteaControl;

import java.io.IOException;
import java.lang.instrument.Instrumentation;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.security.SecureRandom;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Java agent entry point.
 *
 * Start an X-TEA application with:
 *
 *   java -javaagent:xtea-control-agent.jar=port=0,tokenFile=C:\temp\xtea.json \
 *        -jar xteamdl.jar model.xead
 *
 * The agent writes {url, token, pid, ...} to tokenFile. The application itself
 * is not patched or rebuilt.
 */
public final class Agent {
    static final long STARTED_AT = System.currentTimeMillis();
    private static volatile BridgeServer server;

    private Agent() {
    }

    public static void premain(String arguments, Instrumentation instrumentation) {
        start(arguments);
    }

    public static void agentmain(String arguments, Instrumentation instrumentation) {
        start(arguments);
    }

    private static synchronized void start(String arguments) {
        if (server != null) {
            return;
        }
        try {
            Map<String, String> options = parse(arguments);
            String bind = option(options, "bind", "127.0.0.1");
            int port = Integer.parseInt(option(options, "port", "0"));
            boolean allowRemote = Boolean.parseBoolean(option(options, "allowRemote", "false"));
            String token = option(options, "token", "");
            if (token.isEmpty()) {
                token = randomToken();
            }
            Path tokenFile = Paths.get(option(options, "tokenFile",
                    System.getProperty("java.io.tmpdir") + "/xtea-control-" + pid() + ".json"));
            BridgeServer created = new BridgeServer(bind, port, token, allowRemote);
            server = created;
            created.start();
            writeEndpoint(tokenFile, created.endpoint(), token);
            System.err.println("[xtea-control] endpoint=" + created.endpoint());
            System.err.println("[xtea-control] tokenFile=" + tokenFile.toAbsolutePath());
            Runtime.getRuntime().addShutdownHook(new Thread(new Runnable() {
                @Override
                public void run() {
                    BridgeServer current = server;
                    if (current != null) {
                        current.stop();
                    }
                }
            }, "xtea-control-shutdown-hook"));
        } catch (Throwable t) {
            System.err.println("[xtea-control] failed to start: " + describe(t));
            if (t instanceof RuntimeException) {
                // Never prevent the host application from starting because the
                // optional test interface could not bind its port.
            }
        }
    }

    private static void writeEndpoint(Path tokenFile, String endpoint, String token) throws IOException {
        Path parent = tokenFile.toAbsolutePath().getParent();
        if (parent != null) {
            Files.createDirectories(parent);
        }
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("url", endpoint);
        value.put("token", token);
        value.put("pid", Long.valueOf(pid()));
        value.put("protocol", "xtea-control/1");
        value.put("startedAt", Long.valueOf(STARTED_AT));
        Path temporary = tokenFile.resolveSibling(tokenFile.getFileName().toString() + ".tmp");
        Files.write(temporary, Json.write(value).getBytes(StandardCharsets.UTF_8));
        try {
            Files.move(temporary, tokenFile, StandardCopyOption.REPLACE_EXISTING,
                    StandardCopyOption.ATOMIC_MOVE);
        } catch (IOException e) {
            Files.move(temporary, tokenFile, StandardCopyOption.REPLACE_EXISTING);
        }
    }

    private static Map<String, String> parse(String arguments) {
        Map<String, String> options = new LinkedHashMap<String, String>();
        if (arguments == null || arguments.trim().isEmpty()) {
            return options;
        }
        for (String part : arguments.split(",")) {
            String token = part.trim();
            if (token.isEmpty()) {
                continue;
            }
            int separator = token.indexOf('=');
            if (separator < 0) {
                options.put(token, "true");
            } else {
                options.put(token.substring(0, separator).trim(), token.substring(separator + 1).trim());
            }
        }
        return options;
    }

    private static String option(Map<String, String> options, String key, String fallback) {
        String value = options.get(key);
        return value == null || value.isEmpty() ? fallback : value;
    }

    private static String randomToken() {
        byte[] bytes = new byte[24];
        new SecureRandom().nextBytes(bytes);
        StringBuilder value = new StringBuilder();
        for (byte b : bytes) {
            value.append(String.format("%02x", b & 0xff));
        }
        return value.toString();
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

    private static String describe(Throwable throwable) {
        String message = throwable.getMessage();
        return throwable.getClass().getSimpleName() + (message == null ? "" : ": " + message);
    }
}
