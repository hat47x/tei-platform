import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.Locale;

// Test-only process implementation. This line-oriented carrier is NOT a TEI ABI.
public final class NeutralAsciiUpperProvider {
    public static void main(String[] args) throws Exception {
        BufferedReader input = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8));
        String value = input.readLine();
        if (value == null || !value.matches("[a-z]{1,32}") || input.readLine() != null) {
            System.exit(2);
        }
        System.out.println(value.toUpperCase(Locale.ROOT));
    }
}
