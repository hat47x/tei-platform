package xteaControl;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/** Bounded in-process action trace. */
final class Trace {
    private static final int LIMIT = 2000;
    private static final List<Map<String, Object>> EVENTS = new ArrayList<Map<String, Object>>();
    private static long nextId = 1;

    private Trace() {
    }

    static synchronized void record(String type, String target, String detail) {
        Map<String, Object> event = new LinkedHashMap<String, Object>();
        event.put("id", nextId++);
        event.put("time", System.currentTimeMillis());
        event.put("type", type);
        event.put("target", target);
        event.put("detail", detail);
        EVENTS.add(event);
        while (EVENTS.size() > LIMIT) {
            EVENTS.remove(0);
        }
    }

    static synchronized Map<String, Object> since(long id) {
        List<Object> selected = new ArrayList<Object>();
        for (Map<String, Object> event : EVENTS) {
            if (((Number) event.get("id")).longValue() > id) {
                selected.add(event);
            }
        }
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("events", selected);
        value.put("lastId", nextId - 1);
        return value;
    }
}
