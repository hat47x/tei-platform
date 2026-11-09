import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import org.apache.catalina.Context;
import org.apache.catalina.startup.Tomcat;

/** Deploys the XeadServer WAR directory to embedded Tomcat. */
public final class TomcatRunner {
    private TomcatRunner() {
    }

    public static void main(String[] args) throws Exception {
        String webapp = null;
        String endpointFile = null;
        int port = 0;
        for (int i = 0; i < args.length; i++) {
            if ("--webapp".equals(args[i]) && i + 1 < args.length) {
                webapp = args[++i];
            } else if ("--endpoint".equals(args[i]) && i + 1 < args.length) {
                endpointFile = args[++i];
            } else if ("--port".equals(args[i]) && i + 1 < args.length) {
                port = Integer.parseInt(args[++i]);
            }
        }
        if (webapp == null) {
            throw new IllegalArgumentException("--webapp is required");
        }

        Tomcat tomcat = new Tomcat();
        tomcat.setBaseDir(Files.createTempDirectory("xtea-tomcat-").toString());
        tomcat.setPort(port);
        tomcat.getConnector().setProperty("address", "127.0.0.1");
        Context context = tomcat.addWebapp("/xteaServer", Paths.get(webapp).toAbsolutePath().toString());
        context.setReloadable(false);
        tomcat.start();

        String endpoint = "http://127.0.0.1:" + tomcat.getConnector().getLocalPort() + "/xteaServer";
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
                    tomcat.stop();
                } catch (Exception e) {
                    // best effort
                }
            }
        }));
        tomcat.getServer().await();
    }
}
