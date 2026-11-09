import java.io.File;
import java.io.FileOutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.server.ServerConnector;
import org.eclipse.jetty.servlet.ServletContextHandler;
import org.eclipse.jetty.servlet.ServletHolder;

/**
 * Minimal embedded-Jetty launcher for the original XeadServer DBMethod servlet.
 *
 * It does not patch XeadServer. The servlet reads the same SystemDefinition
 * .xeaf file and builds its own DBCP DataSource.
 */
public final class XteaServerHarness {
    private XteaServerHarness() {
    }

    public static void main(String[] args) throws Exception {
        String xeaf = null;
        String endpointFile = null;
        int port = 0;
        for (int i = 0; i < args.length; i++) {
            if ("--xeaf".equals(args[i]) && i + 1 < args.length) {
                xeaf = args[++i];
            } else if ("--endpoint".equals(args[i]) && i + 1 < args.length) {
                endpointFile = args[++i];
            } else if ("--port".equals(args[i]) && i + 1 < args.length) {
                port = Integer.parseInt(args[++i]);
            }
        }
        if (xeaf == null) {
            throw new IllegalArgumentException("--xeaf is required");
        }
        if (!new File(xeaf).isFile()) {
            throw new IllegalArgumentException("xeaf not found: " + xeaf);
        }

        Server server = new Server();
        ServerConnector connector = new ServerConnector(server);
        connector.setHost("127.0.0.1");
        connector.setPort(port);
        server.addConnector(connector);

        ServletContextHandler context = new ServletContextHandler(ServletContextHandler.SESSIONS);
        context.setContextPath("/xteaServer");
        ServletHolder db = new ServletHolder(new xeadServer.DBMethod());
        db.setInitParameter("SystemDefinition", xeaf);
        db.setInitParameter("DataSource", "SalesDs");
        db.setInitParameter("SubDB", "");
        context.addServlet(db, "/DBMethod");

        ServletHolder service = new ServletHolder(new xeadServer.Service());
        service.setInitParameter("SystemDefinition", xeaf);
        service.setInitParameter("User", "00000");
        service.setInitParameter("Password", "0000000000");
        context.addServlet(service, "/Service");
        server.setHandler(context);

        server.start();
        String endpoint = "http://127.0.0.1:" + connector.getLocalPort() + "/xteaServer";
        if (endpointFile != null) {
            Path path = Paths.get(endpointFile).toAbsolutePath();
            if (path.getParent() != null) {
                Files.createDirectories(path.getParent());
            }
            Files.write(path, ("{\"url\":\"" + endpoint + "\"}\n").getBytes(StandardCharsets.UTF_8));
        }
        System.out.println("READY " + endpoint);
        System.out.flush();
        Runtime.getRuntime().addShutdownHook(new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    server.stop();
                } catch (Exception e) {
                    // best effort on shutdown
                }
            }
        }));
        server.join();
    }
}
