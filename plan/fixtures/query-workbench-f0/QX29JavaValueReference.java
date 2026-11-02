import java.math.BigDecimal;
import java.math.BigInteger;
import java.math.RoundingMode;
import java.text.Normalizer;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * QX-29: isolated Java 21 comparison oracle for SQL/Python value semantics.
 * This tests real JDK behavior, NOT TEI-generated Java, a Provider, or Host.
 */
public final class QX29JavaValueReference {
    private static int passed;

    private static void check(String id, boolean condition) {
        if (!condition) {
            throw new AssertionError("FAIL " + id);
        }
        System.out.println("PASS " + id);
        passed++;
    }

    private static void expectThrows(String id, Class<? extends Throwable> expected, Runnable run) {
        try {
            run.run();
        } catch (Throwable e) {
            check(id, expected.isInstance(e));
            return;
        }
        throw new AssertionError("FAIL " + id + " (no exception)");
    }

    public static void main(String[] args) {
        final BigDecimal d01 = new BigDecimal("0.1");
        final BigDecimal d02 = new BigDecimal("0.2");
        check("QX29-01-exact-decimal-addition", d01.add(d02).compareTo(new BigDecimal("0.3")) == 0);
        check("QX29-02-float-same-input-differs", 0.1d + 0.2d != 0.3d);
        check("QX29-03-float-constructor-vs-decimal-input",
                new BigDecimal(0.1d).compareTo(d01) != 0 && BigDecimal.valueOf(0.1d).equals(d01));

        check("QX29-04-positive-rounding-rule",
                new BigDecimal("2.5").setScale(0, RoundingMode.HALF_EVEN).equals(new BigDecimal("2"))
                && new BigDecimal("2.5").setScale(0, RoundingMode.HALF_UP).equals(new BigDecimal("3")));
        check("QX29-05-negative-rounding-rule",
                new BigDecimal("-2.5").setScale(0, RoundingMode.HALF_EVEN).equals(new BigDecimal("-2"))
                && new BigDecimal("-2.5").setScale(0, RoundingMode.HALF_UP).equals(new BigDecimal("-3")));

        final BigDecimal x = new BigDecimal("1.005");
        final BigDecimal roundedBeforeSum = x.setScale(2, RoundingMode.HALF_EVEN).multiply(new BigDecimal("2"));
        final BigDecimal roundedAfterSum = x.multiply(new BigDecimal("2")).setScale(2, RoundingMode.HALF_EVEN);
        check("QX29-06-round-before-or-after-aggregate",
                roundedBeforeSum.equals(new BigDecimal("2.00"))
                && roundedAfterSum.equals(new BigDecimal("2.01")));

        expectThrows("QX29-07-fixed-width-overflow", ArithmeticException.class,
                () -> Math.addExact(Long.MAX_VALUE, 1L));
        check("QX29-08-unbounded-integer-positive",
                BigInteger.valueOf(Long.MAX_VALUE).add(BigInteger.ONE)
                    .equals(new BigInteger("9223372036854775808")));

        expectThrows("QX29-09-adapter-invalid-numeric-text", NumberFormatException.class,
                () -> new BigDecimal("quantity-not-a-number"));

        final Map<String, Object> missing = new HashMap<>();
        final Map<String, Object> explicitNull = new HashMap<>();
        explicitNull.put("quantity", null);
        check("QX29-10-missing-vs-explicit-null",
                !missing.containsKey("quantity") && explicitNull.containsKey("quantity")
                    && explicitNull.get("quantity") == null);

        final BigDecimal s1 = new BigDecimal("1.0");
        final BigDecimal s2 = new BigDecimal("1.00");
        check("QX29-11-value-equal-scale-distinct",
                s1.compareTo(s2) == 0 && !s1.equals(s2));

        expectThrows("QX29-12-unbounded-division-requires-policy", ArithmeticException.class,
                () -> BigDecimal.ONE.divide(new BigDecimal("3")));
        check("QX29-13-division-with-explicit-rounding",
                BigDecimal.ONE.divide(new BigDecimal("3"), 2, RoundingMode.HALF_EVEN)
                    .equals(new BigDecimal("0.33")));

        check("QX29-14-java-case-insensitive-neq-sqlite-nocase",
                "Ä".equalsIgnoreCase("ä") && !"Ä".equals("ä"));
        check("QX29-15-unicode-normalization-must-be-requested",
                !"é".equals("e\u0301")
                    && Normalizer.normalize("e\u0301", Normalizer.Form.NFC).equals("é"));

        final List<String> nullSort = new ArrayList<>(Arrays.asList(null, "a", "b"));
        nullSort.sort(Comparator.nullsLast(Comparator.naturalOrder()));
        check("QX29-16-null-order-is-explicit", nullSort.equals(Arrays.asList("a", "b", null)));

        expectThrows("QX29-17-integer-exponent-parser-rejection", NumberFormatException.class,
                () -> Long.parseLong("1e2"));
        check("QX29-18-decimal-exponent-parser-positive",
                new BigDecimal("1e2").compareTo(new BigDecimal("100")) == 0);

        final Optional<BigDecimal> emptyWithoutIdentity = List.<BigDecimal>of().stream().reduce(BigDecimal::add);
        final BigDecimal emptyWithIdentity = List.<BigDecimal>of().stream().reduce(BigDecimal.ZERO, BigDecimal::add);
        check("QX29-19-empty-aggregate-choice", emptyWithoutIdentity.isEmpty()
                && emptyWithIdentity.equals(BigDecimal.ZERO));

        final BigDecimal negativeZero = new BigDecimal("-0.00");
        check("QX29-20-negative-zero-representation",
                negativeZero.signum() == 0 && Double.doubleToRawLongBits(-0.0d) != Double.doubleToRawLongBits(0.0d));

        System.out.println("QX-29 RESULT " + passed + "/20 PASS (isolated JDK 21 reference)");
    }
}
