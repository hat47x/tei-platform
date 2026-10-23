package xteaServerHarness;

import java.sql.DriverManager;
import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;

/**
 * Registers the embedded Derby driver with DriverManager.
 *
 * Tomcat's webapp classloader can load derby.jar from WEB-INF/lib, but
 * DriverManager does not discover webapp drivers automatically in a WAR
 * deployment. Registering the driver from the same classloader makes
 * DBMethod's DriverManager.getConnection call resolve it.
 */
public final class JdbcBootstrap implements ServletContextListener {
    @Override
    public void contextInitialized(ServletContextEvent event) {
        try {
            Class<?> driverClass = Class.forName("org.apache.derby.jdbc.EmbeddedDriver");
            DriverManager.registerDriver((java.sql.Driver) driverClass.getDeclaredConstructor().newInstance());
        } catch (Exception e) {
            throw new IllegalStateException("failed to register Derby driver", e);
        }
    }

    @Override
    public void contextDestroyed(ServletContextEvent event) {
        // The container owns the classloader and the registered driver.
    }
}
