import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;

/** Creates a disposable H2 schema for the XeadServer DBMethod live test. */
public final class H2Init {
    private H2Init() {
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            throw new IllegalArgumentException("jdbc URL is required");
        }
        Class.forName("org.h2.Driver");
        try (Connection connection = DriverManager.getConnection(args[0], "sa", "");
                Statement statement = connection.createStatement()) {
            statement.execute("DROP TABLE IF EXISTS T");
            statement.execute("CREATE TABLE T (ID INT PRIMARY KEY, NAME VARCHAR(64))");
            statement.execute("DROP ALIAS IF EXISTS SQUARE");
            statement.execute("CREATE ALIAS SQUARE FOR \"java.lang.Math.sqrt\"");
        }
    }
}
