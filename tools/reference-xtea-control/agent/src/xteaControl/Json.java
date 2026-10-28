package xteaControl;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Small dependency-free JSON reader/writer.
 *
 * Only the value shapes needed by the control protocol are supported:
 * object, array, string, number, boolean, and null.
 */
final class Json {
    private Json() {
    }

    static Object parse(String text) {
        if (text == null) {
            throw new IllegalArgumentException("empty JSON");
        }
        Parser parser = new Parser(text);
        Object value = parser.readValue();
        parser.skipWhitespace();
        if (!parser.atEnd()) {
            throw new IllegalArgumentException("trailing content at offset " + parser.offset);
        }
        return value;
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> parseObject(String text) {
        Object value = parse(text);
        if (!(value instanceof Map)) {
            throw new IllegalArgumentException("JSON object required");
        }
        return (Map<String, Object>) value;
    }

    static String write(Object value) {
        StringBuilder out = new StringBuilder();
        writeValue(out, value);
        return out.toString();
    }

    private static void writeValue(StringBuilder out, Object value) {
        if (value == null) {
            out.append("null");
        } else if (value instanceof String) {
            writeString(out, (String) value);
        } else if (value instanceof Boolean) {
            out.append(value.toString());
        } else if (value instanceof Double || value instanceof Float) {
            double number = ((Number) value).doubleValue();
            if (Double.isNaN(number) || Double.isInfinite(number)) {
                out.append("null");
            } else {
                out.append(Double.toString(number));
            }
        } else if (value instanceof Number) {
            out.append(value.toString());
        } else if (value instanceof Map) {
            out.append('{');
            boolean first = true;
            for (Object entryObject : ((Map<?, ?>) value).entrySet()) {
                Map.Entry<?, ?> entry = (Map.Entry<?, ?>) entryObject;
                if (!first) {
                    out.append(',');
                }
                first = false;
                writeString(out, String.valueOf(entry.getKey()));
                out.append(':');
                writeValue(out, entry.getValue());
            }
            out.append('}');
        } else if (value instanceof Iterable) {
            out.append('[');
            boolean first = true;
            for (Object item : (Iterable<?>) value) {
                if (!first) {
                    out.append(',');
                }
                first = false;
                writeValue(out, item);
            }
            out.append(']');
        } else if (value.getClass().isArray()) {
            out.append('[');
            int length = java.lang.reflect.Array.getLength(value);
            for (int i = 0; i < length; i++) {
                if (i > 0) {
                    out.append(',');
                }
                writeValue(out, java.lang.reflect.Array.get(value, i));
            }
            out.append(']');
        } else {
            writeString(out, String.valueOf(value));
        }
    }

    private static void writeString(StringBuilder out, String value) {
        out.append('"');
        for (int i = 0; i < value.length(); i++) {
            char ch = value.charAt(i);
            switch (ch) {
                case '"':
                    out.append("\\\"");
                    break;
                case '\\':
                    out.append("\\\\");
                    break;
                case '\b':
                    out.append("\\b");
                    break;
                case '\f':
                    out.append("\\f");
                    break;
                case '\n':
                    out.append("\\n");
                    break;
                case '\r':
                    out.append("\\r");
                    break;
                case '\t':
                    out.append("\\t");
                    break;
                default:
                    if (ch < 0x20 || ch == 0x2028 || ch == 0x2029) {
                        out.append(String.format("\\u%04x", (int) ch));
                    } else {
                        out.append(ch);
                    }
            }
        }
        out.append('"');
    }

    static Map<String, Object> object(Object... pairs) {
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        for (int i = 0; i + 1 < pairs.length; i += 2) {
            value.put(String.valueOf(pairs[i]), pairs[i + 1]);
        }
        return value;
    }

    private static final class Parser {
        private final String text;
        private int offset;

        Parser(String text) {
            this.text = text;
        }

        boolean atEnd() {
            return offset >= text.length();
        }

        void skipWhitespace() {
            while (offset < text.length()) {
                char ch = text.charAt(offset);
                if (ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n') {
                    offset++;
                } else {
                    break;
                }
            }
        }

        Object readValue() {
            skipWhitespace();
            if (atEnd()) {
                throw error("value expected");
            }
            char ch = text.charAt(offset);
            switch (ch) {
                case '{':
                    return readObject();
                case '[':
                    return readArray();
                case '"':
                    return readString();
                case 't':
                    expect("true");
                    return Boolean.TRUE;
                case 'f':
                    expect("false");
                    return Boolean.FALSE;
                case 'n':
                    expect("null");
                    return null;
                default:
                    if (ch == '-' || (ch >= '0' && ch <= '9')) {
                        return readNumber();
                    }
                    throw error("unexpected character '" + ch + "'");
            }
        }

        private Map<String, Object> readObject() {
            Map<String, Object> value = new LinkedHashMap<String, Object>();
            offset++;
            skipWhitespace();
            if (peek('}')) {
                offset++;
                return value;
            }
            while (true) {
                skipWhitespace();
                String key = readString();
                skipWhitespace();
                if (!peek(':')) {
                    throw error("':' expected");
                }
                offset++;
                value.put(key, readValue());
                skipWhitespace();
                if (peek('}')) {
                    offset++;
                    return value;
                }
                if (!peek(',')) {
                    throw error("',' or '}' expected");
                }
                offset++;
            }
        }

        private List<Object> readArray() {
            List<Object> value = new ArrayList<Object>();
            offset++;
            skipWhitespace();
            if (peek(']')) {
                offset++;
                return value;
            }
            while (true) {
                value.add(readValue());
                skipWhitespace();
                if (peek(']')) {
                    offset++;
                    return value;
                }
                if (!peek(',')) {
                    throw error("',' or ']' expected");
                }
                offset++;
            }
        }

        private String readString() {
            if (!peek('"')) {
                throw error("string expected");
            }
            offset++;
            StringBuilder value = new StringBuilder();
            while (!atEnd()) {
                char ch = text.charAt(offset++);
                if (ch == '"') {
                    return value.toString();
                }
                if (ch != '\\') {
                    value.append(ch);
                    continue;
                }
                if (atEnd()) {
                    throw error("unterminated escape");
                }
                char escaped = text.charAt(offset++);
                switch (escaped) {
                    case '"':
                        value.append('"');
                        break;
                    case '\\':
                        value.append('\\');
                        break;
                    case '/':
                        value.append('/');
                        break;
                    case 'b':
                        value.append('\b');
                        break;
                    case 'f':
                        value.append('\f');
                        break;
                    case 'n':
                        value.append('\n');
                        break;
                    case 'r':
                        value.append('\r');
                        break;
                    case 't':
                        value.append('\t');
                        break;
                    case 'u':
                        if (offset + 4 > text.length()) {
                            throw error("short unicode escape");
                        }
                        String hex = text.substring(offset, offset + 4);
                        offset += 4;
                        try {
                            value.append((char) Integer.parseInt(hex, 16));
                        } catch (NumberFormatException e) {
                            throw error("invalid unicode escape");
                        }
                        break;
                    default:
                        throw error("invalid escape '\\" + escaped + "'");
                }
            }
            throw error("unterminated string");
        }

        private Number readNumber() {
            int start = offset;
            if (peek('-')) {
                offset++;
            }
            while (!atEnd() && Character.isDigit(text.charAt(offset))) {
                offset++;
            }
            boolean fractional = false;
            if (peek('.')) {
                fractional = true;
                offset++;
                while (!atEnd() && Character.isDigit(text.charAt(offset))) {
                    offset++;
                }
            }
            if (peek('e') || peek('E')) {
                fractional = true;
                offset++;
                if (peek('+') || peek('-')) {
                    offset++;
                }
                while (!atEnd() && Character.isDigit(text.charAt(offset))) {
                    offset++;
                }
            }
            String token = text.substring(start, offset);
            try {
                if (fractional) {
                    return Double.valueOf(token);
                }
                return Long.valueOf(token);
            } catch (NumberFormatException e) {
                throw error("invalid number '" + token + "'");
            }
        }

        private void expect(String token) {
            if (!text.startsWith(token, offset)) {
                throw error("'" + token + "' expected");
            }
            offset += token.length();
        }

        private boolean peek(char ch) {
            return !atEnd() && text.charAt(offset) == ch;
        }

        private IllegalArgumentException error(String message) {
            return new IllegalArgumentException(message + " at offset " + offset);
        }
    }
}
