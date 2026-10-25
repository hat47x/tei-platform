package xteaControl;

import java.awt.Color;
import java.awt.Component;
import java.awt.Container;
import java.awt.Dialog;
import java.awt.Frame;
import java.awt.Graphics2D;
import java.awt.KeyboardFocusManager;
import java.awt.Rectangle;
import java.awt.Window;
import java.awt.event.ActionEvent;
import java.awt.event.KeyEvent;
import java.awt.event.WindowEvent;
import java.awt.image.BufferedImage;
import java.io.ByteArrayOutputStream;
import java.lang.reflect.Field;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Deque;
import java.util.HashMap;
import java.util.IdentityHashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.FutureTask;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.regex.Pattern;
import java.util.regex.PatternSyntaxException;
import javax.imageio.ImageIO;
import javax.swing.AbstractButton;
import javax.swing.Action;
import javax.swing.InputMap;
import javax.swing.JCheckBox;
import javax.swing.JComboBox;
import javax.swing.JComponent;
import javax.swing.JDialog;
import javax.swing.JEditorPane;
import javax.swing.JFrame;
import javax.swing.JLabel;
import javax.swing.JList;
import javax.swing.JMenu;
import javax.swing.JMenuBar;
import javax.swing.JPasswordField;
import javax.swing.JPopupMenu;
import javax.swing.JRadioButton;
import javax.swing.RootPaneContainer;
import javax.swing.JSlider;
import javax.swing.JSpinner;
import javax.swing.JTabbedPane;
import javax.swing.JTable;
import javax.swing.JTextArea;
import javax.swing.JTextField;
import javax.swing.JTree;
import javax.swing.KeyStroke;
import javax.swing.SwingUtilities;
import javax.swing.text.JTextComponent;
import javax.swing.tree.TreePath;

/**
 * Swing introspection and input primitives used by the HTTP control surface.
 *
 * Every method that touches a Swing component runs on the Event Dispatch
 * Thread. Read operations time out instead of blocking the caller forever if
 * the application itself has blocked the EDT.
 */
final class Ui {
    static final int DEFAULT_TIMEOUT_MS = 5000;
    private static final int DEFAULT_MAX_NODES = 4000;
    private static final int DEFAULT_MAX_DEPTH = 16;
    private static final int MAX_TEXT = 240;

    private Ui() {
    }

    static <T> T call(Callable<T> callable, long timeoutMs) throws Exception {
        if (SwingUtilities.isEventDispatchThread()) {
            return callable.call();
        }
        FutureTask<T> task = new FutureTask<T>(callable);
        SwingUtilities.invokeLater(task);
        try {
            return task.get(timeoutMs <= 0 ? DEFAULT_TIMEOUT_MS : timeoutMs, TimeUnit.MILLISECONDS);
        } catch (TimeoutException e) {
            task.cancel(true);
            throw new BridgeException("EDT_TIMEOUT", "Swing event dispatch thread did not answer");
        } catch (ExecutionException e) {
            Throwable cause = e.getCause();
            if (cause instanceof BridgeException) {
                throw (BridgeException) cause;
            }
            throw new BridgeException("UI_FAILURE", describe(cause));
        }
    }

    static Map<String, Object> windows() throws Exception {
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                List<Object> list = new ArrayList<Object>();
                int index = 0;
                for (Window window : orderedWindows()) {
                    Map<String, Object> item = windowSummary(window, index++);
                    list.add(item);
                }
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("windows", list);
                value.put("activeTitle", titleOf(activeWindow()));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> snapshot(Map<String, Object> request) throws Exception {
        final int maxNodes = intValue(request.get("maxNodes"), DEFAULT_MAX_NODES);
        final int maxDepth = intValue(request.get("maxDepth"), DEFAULT_MAX_DEPTH);
        final boolean includeTableRows = boolValue(request.get("includeTableRows"), false);
        final String windowFilter = stringValue(request.get("window"));
        final boolean includeInvisible = boolValue(request.get("includeInvisible"), false);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                List<Object> roots = new ArrayList<Object>();
                int rootIndex = 0;
                boolean truncated = false;
                for (Window window : orderedWindows()) {
                    if (windowFilter != null && !matches(windowFilter, titleOf(window))) {
                        continue;
                    }
                    Map<String, Object> root = snapshotWindow(window, rootIndex, maxDepth, maxNodes,
                            includeTableRows, includeInvisible);
                    if (Boolean.TRUE.equals(root.get("truncated"))) {
                        truncated = true;
                    }
                    rootIndex++;
                    roots.add(root);
                }
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("windows", roots);
                value.put("activeTitle", titleOf(activeWindow()));
                value.put("truncated", Boolean.valueOf(truncated));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> menuCatalog() throws Exception {
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                List<Object> menus = new ArrayList<Object>();
                List<Object> actions = new ArrayList<Object>();
                int rootIndex = 0;
                for (Window window : orderedWindows()) {
                    JMenuBar bar = menuBarOf(window);
                    if (bar != null) {
                        collectMenu(bar, 0, window, rootIndex, "m", menus, actions);
                    }
                    collectPopupMenus(window.getComponents(), window, rootIndex, menus, actions, new IdentityHashMap<Object, Object>());
                    rootIndex++;
                }
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("menus", menus);
                value.put("actions", actions);
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> invoke(final Map<String, Object> selector, final long settleMs) throws Exception {
        final Component target = resolve(selector);
        final String label = describeComponent(target);
        final long started = System.currentTimeMillis();
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                try {
                    if (target instanceof AbstractButton) {
                        ((AbstractButton) target).doClick(0);
                    } else if (target instanceof JTabbedPane) {
                        JTabbedPane tabs = (JTabbedPane) target;
                        if (tabs.getTabCount() > 0) {
                            tabs.setSelectedIndex(tabs.getSelectedIndex());
                        }
                    } else {
                        throw new IllegalStateException("component does not accept invoke: " + target.getClass().getName());
                    }
                } catch (Throwable t) {
                    Trace.record("invoke.error", label, describe(t));
                }
            }
        });
        Trace.record("invoke", label, "settleMs=" + settleMs);
        if (settleMs > 0) {
            Thread.sleep(Math.min(settleMs, 30000));
        }
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("accepted", Boolean.TRUE);
        value.put("target", label);
        value.put("startedAt", started);
        value.put("windows", windowTitles());
        return value;
    }

    static Map<String, Object> setValue(final Map<String, Object> selector, final Object requestedValue,
            final String mode) throws Exception {
        final Component target = resolve(selector);
        final String label = describeComponent(target);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                Object oldValue = readValue(target);
                boolean customField = setCustomFieldValue(target, requestedValue);
                if (customField) {
                    // X-TEA wraps editable fields in XFTextField and similar
                    // components; setting the inner JTextField alone leaves the
                    // wrapper's value state unchanged.
                } else if (target instanceof JTextComponent) {
                    JTextComponent text = (JTextComponent) target;
                    String value = requestedValue == null ? "" : String.valueOf(requestedValue);
                    if ("append".equalsIgnoreCase(mode)) {
                        text.setText(text.getText() + value);
                    } else {
                        text.setText(value);
                    }
                } else if (target instanceof JComboBox) {
                    JComboBox<?> combo = (JComboBox<?>) target;
                    selectCombo(combo, requestedValue);
                } else if (target instanceof JCheckBox) {
                    ((JCheckBox) target).setSelected(boolValue(requestedValue, false));
                } else if (target instanceof JRadioButton) {
                    ((JRadioButton) target).setSelected(boolValue(requestedValue, true));
                } else if (target instanceof JSpinner) {
                    JSpinner spinner = (JSpinner) target;
                    Object value = requestedValue;
                    if (requestedValue instanceof Number) {
                        value = ((Number) requestedValue).intValue();
                    }
                    spinner.setValue(value);
                } else if (target instanceof JSlider) {
                    ((JSlider) target).setValue(intValue(requestedValue, 0));
                } else {
                    throw new BridgeException("UNSUPPORTED_COMPONENT",
                            "component does not accept set: " + target.getClass().getName());
                }
                Trace.record("set", label, "from=" + brief(oldValue) + " to=" + brief(readValue(target)));
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("target", label);
                value.put("oldValue", oldValue);
                value.put("value", readValue(target));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> select(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        final String label = describeComponent(target);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                if (target instanceof JTable) {
                    JTable table = (JTable) target;
                    int row = intValue(request.get("row"), -1);
                    int column = intValue(request.get("column"), 0);
                    if (row < 0 || row >= table.getRowCount()) {
                        throw new BridgeException("INVALID_ROW", "row out of range: " + row);
                    }
                    if (column < 0 || column >= table.getColumnCount()) {
                        throw new BridgeException("INVALID_COLUMN", "column out of range: " + column);
                    }
                    table.setRowSelectionInterval(row, row);
                    table.setColumnSelectionInterval(column, column);
                    table.scrollRectToVisible(table.getCellRect(row, column, true));
                } else if (target instanceof JTree) {
                    JTree tree = (JTree) target;
                    TreePath path = treePath(tree, request.get("path"), intValue(request.get("row"), -1));
                    tree.setSelectionPath(path);
                    tree.scrollPathToVisible(path);
                } else if (target instanceof JList) {
                    JList<?> list = (JList<?>) target;
                    int index = intValue(request.get("index"), intValue(request.get("row"), -1));
                    if (index < 0 || index >= list.getModel().getSize()) {
                        throw new BridgeException("INVALID_INDEX", "list index out of range: " + index);
                    }
                    list.setSelectedIndex(index);
                    list.ensureIndexIsVisible(index);
                } else if (target instanceof JTabbedPane) {
                    JTabbedPane tabs = (JTabbedPane) target;
                    int index = intValue(request.get("index"), -1);
                    String title = stringValue(request.get("title"));
                    if (title != null) {
                        index = indexOfTab(tabs, title);
                    }
                    if (index < 0 || index >= tabs.getTabCount()) {
                        throw new BridgeException("INVALID_INDEX", "tab index out of range: " + index);
                    }
                    tabs.setSelectedIndex(index);
                } else {
                    throw new BridgeException("UNSUPPORTED_COMPONENT",
                            "component does not accept select: " + target.getClass().getName());
                }
                Trace.record("select", label, Json.write(request));
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("target", label);
                value.put("selected", readValue(target));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> treeRead(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        if (!(target instanceof JTree)) {
            throw new BridgeException("NOT_A_TREE", "selector does not resolve to a tree");
        }
        final JTree tree = (JTree) target;
        final int maxRows = Math.min(intValue(request.get("maxRows"), 500), 2000);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                List<Object> rows = new ArrayList<Object>();
                Object root = tree.getModel().getRoot();
                collectTreeRows(tree, new TreePath(root), 0, rows, maxRows);
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("rows", rows);
                value.put("rowCount", rows.size());
                value.put("selectedPath", tree.getSelectionPath() == null ? null : tree.getSelectionPath().toString());
                value.put("truncated", rows.size() >= maxRows);
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    private static void collectTreeRows(JTree tree, TreePath path, int depth, List<Object> rows, int maxRows) {
        if (rows.size() >= maxRows) {
            return;
        }
        Object value = path.getLastPathComponent();
        int visibleRow = tree.getRowForPath(path);
        Map<String, Object> entry = new LinkedHashMap<String, Object>();
        entry.put("row", visibleRow);
        entry.put("depth", depth);
        entry.put("indices", treeIndices(tree, path));
        entry.put("text", treeText(tree, path, visibleRow));
        entry.put("leaf", Boolean.valueOf(tree.getModel().isLeaf(value)));
        entry.put("expanded", Boolean.valueOf(visibleRow >= 0 && tree.isExpanded(visibleRow)));
        rows.add(entry);
        int count = tree.getModel().getChildCount(value);
        for (int i = 0; i < count; i++) {
            Object child = tree.getModel().getChild(value, i);
            collectTreeRows(tree, path.pathByAddingChild(child), depth + 1, rows, maxRows);
        }
    }

    private static List<Object> treeIndices(JTree tree, TreePath path) {
        List<Object> indices = new ArrayList<Object>();
        Object parent = tree.getModel().getRoot();
        indices.add(Integer.valueOf(0));
        Object[] components = path.getPath();
        for (int i = 1; i < components.length; i++) {
            int index = tree.getModel().getIndexOfChild(parent, components[i]);
            indices.add(Integer.valueOf(index));
            parent = components[i];
        }
        return indices;
    }

    private static String treeText(JTree tree, TreePath path, int row) {
        Object value = path.getLastPathComponent();
        if (row < 0) {
            return String.valueOf(value);
        }
        try {
            return tree.convertValueToText(value, tree.isRowSelected(row), tree.isExpanded(row),
                    tree.getModel().isLeaf(value), row, false);
        } catch (RuntimeException e) {
            return String.valueOf(value);
        }
    }

    static Map<String, Object> tableSet(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        if (!(target instanceof JTable)) {
            throw new BridgeException("NOT_A_TABLE", "selector does not resolve to a table");
        }
        final JTable table = (JTable) target;
        final int row = intValue(request.get("row"), -1);
        final Object columnSpec = request.get("column");
        final Object cellValue = request.get("value");
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                if (row < 0 || row >= table.getRowCount()) {
                    throw new BridgeException("INVALID_ROW", "row out of range: " + row);
                }
                int column = columnIndex(table, columnSpec);
                if (column < 0 || column >= table.getColumnCount()) {
                    throw new BridgeException("INVALID_COLUMN", "column out of range: " + column);
                }
                if (!table.isCellEditable(row, column)) {
                    throw new BridgeException("CELL_NOT_EDITABLE", "cell is not editable");
                }
                table.setValueAt(cellValue, row, column);
                if (table.isEditing()) {
                    table.getCellEditor().stopCellEditing();
                }
                Trace.record("table.set", describeComponent(table), "row=" + row + " column=" + column
                        + " value=" + brief(cellValue));
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("row", row);
                value.put("column", column);
                value.put("value", table.getValueAt(row, column));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> tableInfo(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        if (!(target instanceof JTable)) {
            throw new BridgeException("NOT_A_TABLE", "selector does not resolve to a table");
        }
        final JTable table = (JTable) target;
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                List<Object> columns = new ArrayList<Object>();
                for (int i = 0; i < table.getColumnCount(); i++) {
                    columns.add(table.getColumnName(i));
                }
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("tableRowCount", table.getRowCount());
                value.put("modelRowCount", table.getModel().getRowCount());
                value.put("modelClass", table.getModel().getClass().getName());
                value.put("columns", columns);
                value.put("selectedRow", table.getSelectedRow());
                value.put("selectedColumn", table.getSelectedColumn());
                value.put("rowSorterClass", table.getRowSorter() == null ? null
                        : table.getRowSorter().getClass().getName());
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> tableRead(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        if (!(target instanceof JTable)) {
            throw new BridgeException("NOT_A_TABLE", "selector does not resolve to a table");
        }
        final JTable table = (JTable) target;
        final int maxRows = Math.min(intValue(request.get("maxRows"), 50), 500);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                Map<String, Object> value = readCustomTable(table, maxRows);
                if (value != null) {
                    return value;
                }
                List<Object> columns = new ArrayList<Object>();
                for (int c = 0; c < table.getColumnCount(); c++) {
                    columns.add(table.getColumnName(c));
                }
                List<Object> rows = new ArrayList<Object>();
                for (int r = 0; r < table.getRowCount() && r < maxRows; r++) {
                    List<Object> row = new ArrayList<Object>();
                    for (int c = 0; c < table.getColumnCount(); c++) {
                        row.add(brief(table.getValueAt(r, c)));
                    }
                    rows.add(row);
                }
                value = new LinkedHashMap<String, Object>();
                value.put("columns", columns);
                value.put("rows", rows);
                value.put("rowCount", table.getRowCount());
                value.put("selectedRow", table.getSelectedRow());
                value.put("truncated", table.getRowCount() > maxRows);
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> key(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = selector == null || selector.isEmpty() ? null : resolve(selector);
        final String keyName = stringValue(request.get("key"));
        final String text = stringValue(request.get("text"));
        final int modifiers = modifiers(request.get("modifiers"));
        final String label = target == null ? "focus-owner" : describeComponent(target);
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                try {
                    Component receiver = target;
                    if (receiver == null) {
                        receiver = KeyboardFocusManager.getCurrentKeyboardFocusManager().getFocusOwner();
                    }
                    if (receiver == null) {
                        throw new IllegalStateException("no focused component");
                    }
                    if (text != null && !text.isEmpty()) {
                        for (int i = 0; i < text.length(); i++) {
                            char ch = text.charAt(i);
                            receiver.dispatchEvent(new KeyEvent(receiver, KeyEvent.KEY_TYPED,
                                    System.currentTimeMillis(), modifiers, KeyEvent.VK_UNDEFINED, ch));
                        }
                    }
                    if (keyName != null && !keyName.isEmpty()) {
                        int keyCode = keyCode(keyName);
                        boolean hasKeyListener = receiver.getKeyListeners().length > 0;
                        receiver.dispatchEvent(new KeyEvent(receiver, KeyEvent.KEY_PRESSED,
                                System.currentTimeMillis(), modifiers, keyCode, KeyEvent.CHAR_UNDEFINED));
                        receiver.dispatchEvent(new KeyEvent(receiver, KeyEvent.KEY_RELEASED,
                                System.currentTimeMillis(), modifiers, keyCode, KeyEvent.CHAR_UNDEFINED));
                        if (!hasKeyListener) {
                            dispatchKeyBinding(receiver, keyCode, modifiers);
                        }
                    }
                } catch (Throwable t) {
                    Trace.record("key.error", label, describe(t));
                }
            }
        });
        Trace.record("key", label, "key=" + keyName + " text=" + text);
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("accepted", Boolean.TRUE);
        value.put("target", label);
        return value;
    }

    static Map<String, Object> click(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        final int count = Math.max(1, Math.min(intValue(request.get("count"), 1), 3));
        final int row = intValue(request.get("row"), -1);
        final int column = intValue(request.get("column"), -1);
        final int button = intValue(request.get("button"), 1) == 3 ? 3 : 1;
        final String label = describeComponent(target);
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                try {
                    int x = Math.max(1, target.getWidth() / 2);
                    int y = Math.max(1, target.getHeight() / 2);
                    if (target instanceof JTable && row >= 0) {
                        JTable table = (JTable) target;
                        Rectangle cell = table.getCellRect(row, column >= 0 ? column : 0, true);
                        x = cell.x + Math.max(1, cell.width / 2);
                        y = cell.y + Math.max(1, cell.height / 2);
                    } else if (target instanceof JList && row >= 0) {
                        JList list = (JList) target;
                        Rectangle cell = list.getCellBounds(row, row);
                        if (cell != null) {
                            x = cell.x + Math.max(1, cell.width / 2);
                            y = cell.y + Math.max(1, cell.height / 2);
                        }
                    } else if (target instanceof JTree && ((JTree) target).getSelectionPath() != null) {
                        Rectangle path = ((JTree) target).getPathBounds(((JTree) target).getSelectionPath());
                        if (path != null) {
                            x = path.x + Math.max(1, path.width / 2);
                            y = path.y + Math.max(1, path.height / 2);
                        }
                    }
                    long now = System.currentTimeMillis();
                    int mask = button == 3 ? java.awt.event.InputEvent.BUTTON3_DOWN_MASK
                            : java.awt.event.InputEvent.BUTTON1_DOWN_MASK;
                    for (int i = 1; i <= count; i++) {
                        target.dispatchEvent(new java.awt.event.MouseEvent(target,
                                java.awt.event.MouseEvent.MOUSE_PRESSED, now, mask, x, y, i, false, button));
                        target.dispatchEvent(new java.awt.event.MouseEvent(target,
                                java.awt.event.MouseEvent.MOUSE_RELEASED, now, mask, x, y, i, false, button));
                        if (button == 1) {
                            target.dispatchEvent(new java.awt.event.MouseEvent(target,
                                    java.awt.event.MouseEvent.MOUSE_CLICKED, now, mask, x, y, i, false, button));
                        }
                    }
                } catch (Throwable t) {
                    Trace.record("click.error", label, describe(t));
                }
            }
        });
        Trace.record("click", label, "count=" + count + ",row=" + row + ",column=" + column + ",button=" + button);
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("accepted", Boolean.TRUE);
        value.put("target", label);
        value.put("count", count);
        value.put("button", button);
        return value;
    }

    static Map<String, Object> drag(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        if (!(target instanceof JTable)) {
            throw new BridgeException("NOT_A_TABLE", "drag currently requires a table");
        }
        final JTable table = (JTable) target;
        final int fromRow = intValue(request.get("fromRow"), -1);
        final int toRow = intValue(request.get("toRow"), -1);
        final int column = intValue(request.get("column"), 0);
        if (fromRow < 0 || toRow < 0 || fromRow >= table.getRowCount() || toRow >= table.getRowCount()) {
            throw new BridgeException("INVALID_ARGUMENT", "fromRow/toRow must be valid table rows");
        }
        final String label = describeComponent(target);
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                try {
                    Rectangle from = table.getCellRect(fromRow, column, true);
                    Rectangle to = table.getCellRect(toRow, column, true);
                    int fromX = from.x + from.width / 2;
                    int fromY = from.y + from.height / 2;
                    int toX = to.x + to.width / 2;
                    int toY = to.y + to.height / 2;
                    long now = System.currentTimeMillis();
                    target.dispatchEvent(new java.awt.event.MouseEvent(target,
                            java.awt.event.MouseEvent.MOUSE_PRESSED, now,
                            java.awt.event.InputEvent.BUTTON1_DOWN_MASK, fromX, fromY, 1, false,
                            java.awt.event.MouseEvent.BUTTON1));
                    target.dispatchEvent(new java.awt.event.MouseEvent(target,
                            java.awt.event.MouseEvent.MOUSE_DRAGGED, now,
                            java.awt.event.InputEvent.BUTTON1_DOWN_MASK, toX, toY, 1, false,
                            java.awt.event.MouseEvent.BUTTON1));
                    target.dispatchEvent(new java.awt.event.MouseEvent(target,
                            java.awt.event.MouseEvent.MOUSE_RELEASED, now,
                            java.awt.event.InputEvent.BUTTON1_DOWN_MASK, toX, toY, 1, false,
                            java.awt.event.MouseEvent.BUTTON1));
                } catch (Throwable t) {
                    Trace.record("drag.error", label, describe(t));
                }
            }
        });
        Trace.record("drag", label, "from=" + fromRow + " to=" + toRow);
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("accepted", Boolean.TRUE);
        value.put("target", label);
        value.put("fromRow", fromRow);
        value.put("toRow", toRow);
        return value;
    }

    static Map<String, Object> invokeMethod(final Map<String, Object> selector, final Map<String, Object> request)
            throws Exception {
        final Component target = resolve(selector);
        final String name = String.valueOf(request.get("method"));
        final java.lang.reflect.Method method = findMethod(target.getClass(), name);
        if (method == null) {
            StringBuilder available = new StringBuilder();
            for (java.lang.reflect.Method candidate : target.getClass().getDeclaredMethods()) {
                if (candidate.getName().toLowerCase().contains("delete")) {
                    available.append(candidate.getName()).append(" ");
                }
            }
            throw new BridgeException("METHOD_NOT_FOUND",
                    "no zero-arg or ActionEvent method: " + name
                            + " on " + target.getClass().getName()
                            + " (delete methods: " + available.toString().trim() + ")");
        }
        method.setAccessible(true);
        final String label = describeComponent(target);
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                try {
                    if (method.getParameterTypes().length == 0) {
                        method.invoke(target);
                    } else {
                        method.invoke(target, new ActionEvent(target, ActionEvent.ACTION_PERFORMED, ""));
                    }
                } catch (Throwable t) {
                    Trace.record("invoke-method.error", label, describe(t));
                }
            }
        });
        Trace.record("invoke-method", label, name);
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("accepted", Boolean.TRUE);
        value.put("target", label);
        value.put("method", name);
        return value;
    }

    private static java.lang.reflect.Method findMethod(Class<?> type, String name) {
        for (Class<?> current = type; current != null; current = current.getSuperclass()) {
            for (java.lang.reflect.Method method : current.getDeclaredMethods()) {
                if (!method.getName().equals(name)) {
                    continue;
                }
                Class<?>[] parameters = method.getParameterTypes();
                if (parameters.length == 0
                        || (parameters.length == 1 && parameters[0].equals(ActionEvent.class))) {
                    return method;
                }
            }
        }
        return null;
    }

    static Map<String, Object> close(final Map<String, Object> selector) throws Exception {
        final Component target = resolve(selector);
        final String label = describeComponent(target);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() {
                Window window = target instanceof Window ? (Window) target : SwingUtilities.getWindowAncestor(target);
                if (window == null) {
                    throw new BridgeException("NO_WINDOW", "component has no window");
                }
                if (window instanceof Frame) {
                    window.dispatchEvent(new WindowEvent(window, WindowEvent.WINDOW_CLOSING));
                } else {
                    window.dispose();
                }
                Trace.record("close", label, titleOf(window));
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("closed", titleOf(window));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static Map<String, Object> waitFor(final Map<String, Object> request) throws Exception {
        final long timeoutMs = Math.min(longValue(request.get("timeoutMs"), 10000L), 60000L);
        final long deadline = System.currentTimeMillis() + timeoutMs;
        final String windowPattern = stringValue(request.get("window"));
        final String state = stringValue(request.get("state"));
        final Map<String, Object> selector = mapValue(request.get("selector"));
        Throwable last = null;
        while (System.currentTimeMillis() <= deadline) {
            try {
                if (windowPattern != null) {
                    boolean present = windowPresent(windowPattern);
                    if ("absent".equalsIgnoreCase(state) ? !present : present) {
                        Map<String, Object> value = new LinkedHashMap<String, Object>();
                        value.put("matched", Boolean.TRUE);
                        value.put("window", windowPattern);
                        value.put("present", present);
                        return value;
                    }
                } else if (selector != null) {
                    boolean present = selectorPresent(selector);
                    if ("absent".equalsIgnoreCase(state) ? !present : present) {
                        Map<String, Object> value = new LinkedHashMap<String, Object>();
                        value.put("matched", Boolean.TRUE);
                        value.put("selector", selector);
                        value.put("present", present);
                        return value;
                    }
                } else {
                    throw new BridgeException("INVALID_ARGUMENT", "wait requires window or selector");
                }
            } catch (Throwable t) {
                last = t;
            }
            Thread.sleep(50L);
        }
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("matched", Boolean.FALSE);
        value.put("timedOut", Boolean.TRUE);
        value.put("window", windowPattern);
        value.put("selector", selector);
        if (last != null) {
            value.put("lastError", describe(last));
        }
        return value;
    }

    static Map<String, Object> screenshot(Map<String, Object> selector) throws Exception {
        final Component target = selector == null || selector.isEmpty() ? null : resolve(selector);
        return call(new Callable<Map<String, Object>>() {
            @Override
            public Map<String, Object> call() throws Exception {
                Component component = target;
                if (component == null) {
                    Window active = activeWindow();
                    if (active == null) {
                        throw new BridgeException("NO_WINDOW", "no visible window");
                    }
                    component = active;
                }
                Rectangle bounds = component.getBounds();
                int width = Math.max(1, Math.min(bounds.width, 4096));
                int height = Math.max(1, Math.min(bounds.height, 4096));
                BufferedImage image = new BufferedImage(width, height, BufferedImage.TYPE_INT_ARGB);
                Graphics2D graphics = image.createGraphics();
                try {
                    graphics.setColor(Color.WHITE);
                    graphics.fillRect(0, 0, width, height);
                    if (component instanceof JComponent) {
                        ((JComponent) component).printAll(graphics);
                    } else {
                        component.printAll(graphics);
                    }
                } finally {
                    graphics.dispose();
                }
                ByteArrayOutputStream bytes = new ByteArrayOutputStream();
                ImageIO.write(image, "png", bytes);
                Map<String, Object> value = new LinkedHashMap<String, Object>();
                value.put("target", describeComponent(component));
                value.put("width", width);
                value.put("height", height);
                value.put("pngBase64", java.util.Base64.getEncoder().encodeToString(bytes.toByteArray()));
                return value;
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    static List<Object> windowTitles() {
        List<Object> titles = new ArrayList<Object>();
        for (Window window : Window.getWindows()) {
            if (window.isVisible()) {
                titles.add(titleOf(window));
            }
        }
        return titles;
    }

    static boolean windowPresent(String pattern) throws Exception {
        Pattern compiled = pattern(pattern);
        for (Window window : Window.getWindows()) {
            if (window.isVisible() && compiled.matcher(titleOf(window)).find()) {
                return true;
            }
        }
        return false;
    }

    static boolean selectorPresent(Map<String, Object> selector) throws Exception {
        try {
            resolve(selector);
            return true;
        } catch (BridgeException e) {
            if ("NOT_FOUND".equals(e.code)) {
                return false;
            }
            throw e;
        }
    }

    private static Component resolve(final Map<String, Object> selector) throws Exception {
        if (selector == null || selector.isEmpty()) {
            throw new BridgeException("INVALID_ARGUMENT", "selector is required");
        }
        return call(new Callable<Component>() {
            @Override
            public Component call() {
                return resolveOnEdt(selector);
            }
        }, DEFAULT_TIMEOUT_MS);
    }

    private static Component resolveOnEdt(Map<String, Object> selector) {
        String windowPattern = stringValue(selector.get("window"));
        String path = stringValue(selector.get("path"));
        List<Window> candidates = new ArrayList<Window>();
        Window active = activeWindow();
        if (active != null) {
            candidates.add(active);
        }
        for (Window window : orderedWindows()) {
            if (!candidates.contains(window)) {
                candidates.add(window);
            }
        }
        String role = normalizeRole(stringValue(selector.get("role")));
        Pattern text = optionalPattern(selector.get("text"));
        Pattern contains = optionalPattern(selector.get("textContains"));
        Pattern name = optionalPattern(selector.get("name"));
        Pattern className = optionalPattern(selector.get("class"));
        int wantedIndex = intValue(selector.get("index"), 0);
        int found = 0;
        Component firstMatch = null;
        Component firstVisibleMatch = null;
        for (Window window : candidates) {
            if (windowPattern != null && !matches(windowPattern, titleOf(window))) {
                continue;
            }
            if (path != null) {
                Component component = byPath(window, path);
                if (component != null && matchesComponent(component, role, text, contains, name, className)) {
                    return component;
                }
                continue;
            }
            if ("window".equals(role) && matchesComponent(window, role, text, contains, name, className)) {
                if (found == wantedIndex) {
                    return window;
                }
                found++;
                continue;
            }
            List<Component> components = flatten(window, true);
            for (Component component : components) {
                if (matchesComponent(component, role, text, contains, name, className)) {
                    if (found == wantedIndex) {
                        if (component.isShowing()) {
                            return component;
                        }
                        if (component.isVisible() && firstVisibleMatch == null) {
                            firstVisibleMatch = component;
                        }
                        if (firstMatch == null) {
                            firstMatch = component;
                        }
                    }
                    found++;
                }
            }
        }
        if (firstVisibleMatch != null) {
            return firstVisibleMatch;
        }
        if (firstMatch != null) {
            return firstMatch;
        }
        throw new BridgeException("NOT_FOUND", "no component matched selector " + Json.write(selector));
    }

    private static boolean matchesComponent(Component component, String role, Pattern text, Pattern contains,
            Pattern name, Pattern className) {
        if (role != null && !role.equals(roleOf(component))) {
            return false;
        }
        String label = labelOf(component);
        if (text != null && !text.matcher(label).find()) {
            return false;
        }
        if (contains != null && !contains.matcher(label.toLowerCase()).find()) {
            return false;
        }
        if (name != null && !name.matcher(String.valueOf(component.getName())).find()) {
            return false;
        }
        if (className != null && !className.matcher(component.getClass().getName()).find()) {
            return false;
        }
        return true;
    }

    private static Map<String, Object> snapshotWindow(Window window, int rootIndex, int maxDepth, int maxNodes,
            boolean includeTableRows, boolean includeInvisible) {
        int[] count = new int[] {0};
        Map<String, Object> root = node(window, "w" + rootIndex, "", 0, maxDepth, maxNodes, includeTableRows,
                includeInvisible, count, new IdentityHashMap<Object, Object>());
        root.put("kind", "window");
        root.put("modal", window instanceof Dialog ? Boolean.valueOf(((Dialog) window).isModal()) : Boolean.FALSE);
        root.put("active", Boolean.valueOf(window == activeWindow()));
        if (count[0] >= maxNodes) {
            root.put("truncated", Boolean.TRUE);
        }
        return root;
    }

    private static Map<String, Object> node(Component component, String ref, String path, int depth, int maxDepth,
            int maxNodes, boolean includeTableRows, boolean includeInvisible, int[] count,
            IdentityHashMap<Object, Object> seen) {
        if (count[0] >= maxNodes) {
            return null;
        }
        count[0]++;
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("ref", ref);
        value.put("path", path);
        value.put("role", roleOf(component));
        value.put("class", component.getClass().getName());
        value.put("text", labelOf(component));
        value.put("name", component.getName());
        value.put("enabled", Boolean.valueOf(component.isEnabled()));
        value.put("visible", Boolean.valueOf(component.isVisible()));
        value.put("showing", Boolean.valueOf(component.isShowing()));
        value.put("bounds", bounds(component));
        if (component instanceof JTextComponent) {
            value.put("editable", Boolean.valueOf(((JTextComponent) component).isEditable()));
            value.put("textValue", brief(readValue(component)));
        }
        if (component instanceof AbstractButton) {
            value.put("selected", Boolean.valueOf(((AbstractButton) component).isSelected()));
        }
        if (component instanceof JComboBox) {
            JComboBox<?> combo = (JComboBox<?>) component;
            value.put("selectedIndex", combo.getSelectedIndex());
            value.put("selectedItem", brief(combo.getSelectedItem()));
            List<Object> items = new ArrayList<Object>();
            for (int i = 0; i < combo.getItemCount() && i < 100; i++) {
                items.add(brief(combo.getItemAt(i)));
            }
            value.put("items", items);
        }
        if (component instanceof JTabbedPane) {
            JTabbedPane tabs = (JTabbedPane) component;
            List<Object> titles = new ArrayList<Object>();
            for (int i = 0; i < tabs.getTabCount(); i++) {
                titles.add(tabs.getTitleAt(i));
            }
            value.put("tabs", titles);
            value.put("selectedIndex", tabs.getSelectedIndex());
        }
        if (component instanceof JTable) {
            JTable table = (JTable) component;
            value.put("rowCount", table.getRowCount());
            value.put("columnCount", table.getColumnCount());
            List<Object> columns = new ArrayList<Object>();
            for (int c = 0; c < table.getColumnCount(); c++) {
                columns.add(table.getColumnName(c));
            }
            value.put("columns", columns);
            if (includeTableRows) {
                value.put("rows", tableRows(table, 20));
            }
        }
        if (component instanceof JTree) {
            JTree tree = (JTree) component;
            value.put("rowCount", tree.getRowCount());
            TreePath selection = tree.getSelectionPath();
            value.put("selectedPath", selection == null ? null : selection.toString());
        }
        if (component instanceof JList) {
            JList<?> list = (JList<?>) component;
            value.put("itemCount", list.getModel().getSize());
            value.put("selectedIndex", list.getSelectedIndex());
        }
        if (component instanceof JSpinner) {
            value.put("value", brief(((JSpinner) component).getValue()));
        }
        if (component instanceof JSlider) {
            value.put("value", ((JSlider) component).getValue());
        }
        if (depth >= maxDepth || count[0] >= maxNodes) {
            return value;
        }
        List<Object> children = new ArrayList<Object>();
        if (component instanceof Window) {
            JMenuBar bar = menuBarOf(component);
            if (bar != null) {
                Map<String, Object> child = node(bar, ref + "/m", "m", depth + 1, maxDepth, maxNodes,
                        includeTableRows, includeInvisible, count, seen);
                if (child != null) {
                    children.add(child);
                }
            }
            Container content = contentOf((Window) component);
            if (content != null) {
                Map<String, Object> child = node(content, ref + "/0", "0", depth + 1, maxDepth, maxNodes,
                        includeTableRows, includeInvisible, count, seen);
                if (child != null) {
                    children.add(child);
                }
            }
        } else if (component instanceof JMenu) {
            JMenu menu = (JMenu) component;
            for (int i = 0; i < menu.getItemCount(); i++) {
                Component item = menu.getItem(i);
                if (item == null) {
                    continue;
                }
                String childPath = path.isEmpty() ? String.valueOf(i) : path + "." + i;
                Map<String, Object> child = node(item, ref + "/" + i, childPath, depth + 1, maxDepth, maxNodes,
                        includeTableRows, includeInvisible, count, seen);
                if (child != null) {
                    children.add(child);
                }
            }
        } else if (component instanceof Container) {
            Container container = (Container) component;
            for (int i = 0; i < container.getComponentCount(); i++) {
                Component childComponent = container.getComponent(i);
                if (!includeInvisible && !childComponent.isVisible() && !(childComponent instanceof JMenu)) {
                    continue;
                }
                String childPath = path.isEmpty() ? String.valueOf(i) : path + "." + i;
                Map<String, Object> child = node(childComponent, ref + "/" + i, childPath, depth + 1, maxDepth,
                        maxNodes, includeTableRows, includeInvisible, count, seen);
                if (child != null) {
                    children.add(child);
                }
            }
        }
        JComponent jcomponent = component instanceof JComponent ? (JComponent) component : null;
        if (jcomponent != null && jcomponent.getComponentPopupMenu() != null
                && !seen.containsKey(jcomponent.getComponentPopupMenu())) {
            seen.put(jcomponent.getComponentPopupMenu(), Boolean.TRUE);
            Map<String, Object> popup = node(jcomponent.getComponentPopupMenu(), ref + "/p",
                    path.isEmpty() ? "p" : path + ".p", depth + 1, maxDepth, maxNodes, includeTableRows,
                    includeInvisible, count, seen);
            if (popup != null) {
                children.add(popup);
            }
        }
        if (!children.isEmpty()) {
            value.put("children", children);
        }
        return value;
    }

    private static List<Component> flatten(Window window, boolean includeMenus) {
        List<Component> list = new ArrayList<Component>();
        JMenuBar bar = menuBarOf(window);
        if (bar != null) {
            addAll(bar, list, includeMenus);
        }
        Container content = contentOf(window);
        if (content != null) {
            addAll(content, list, includeMenus);
        }
        if (window instanceof JFrame) {
            JFrame frame = (JFrame) window;
            if (frame.getGlassPane() != null) {
                addAll(frame.getGlassPane(), list, includeMenus);
            }
        }
        return list;
    }

    private static void addAll(Component component, List<Component> list, boolean includeMenus) {
        list.add(component);
        if (component instanceof JMenu) {
            JMenu menu = (JMenu) component;
            for (int i = 0; i < menu.getItemCount(); i++) {
                if (menu.getItem(i) != null) {
                    addAll(menu.getItem(i), list, includeMenus);
                }
            }
        } else if (component instanceof Container) {
            Container container = (Container) component;
            for (int i = 0; i < container.getComponentCount(); i++) {
                Component child = container.getComponent(i);
                if (includeMenus || !(child instanceof JMenu)) {
                    addAll(child, list, includeMenus);
                }
            }
        }
        if (component instanceof JComponent) {
            JComponent jcomponent = (JComponent) component;
            if (jcomponent.getComponentPopupMenu() != null) {
                addAll(jcomponent.getComponentPopupMenu(), list, includeMenus);
            }
        }
    }

    private static Component byPath(Window window, String path) {
        if (path.isEmpty()) {
            return window;
        }
        String[] parts = path.split("\\.");
        if (parts.length == 0) {
            return null;
        }
        Component current;
        int first = 0;
        if (parts[0].equals("m")) {
            current = menuBarOf(window);
            if (current == null) {
                return null;
            }
            first = 1;
        } else if (parts[0].equals("0")) {
            current = contentOf(window);
            if (current == null) {
                return null;
            }
            first = 1;
            if (parts.length == 1) {
                return current;
            }
        } else {
            current = contentOf(window);
        }
        for (int position = first; position < parts.length; position++) {
            String part = parts[position];
            if (part.equals("p")) {
                if (!(current instanceof JComponent)) {
                    return null;
                }
                current = ((JComponent) current).getComponentPopupMenu();
                continue;
            }
            int index;
            try {
                index = Integer.parseInt(part);
            } catch (NumberFormatException e) {
                return null;
            }
            if (current instanceof JMenu) {
                JMenu menu = (JMenu) current;
                if (index >= menu.getItemCount()) {
                    return null;
                }
                current = menu.getItem(index);
            } else if (current instanceof Container) {
                Container container = (Container) current;
                if (index >= container.getComponentCount()) {
                    return null;
                }
                current = container.getComponent(index);
            } else {
                return null;
            }
        }
        return current;
    }

    private static List<Window> orderedWindows() {
        List<Window> windows = new ArrayList<Window>();
        Window active = activeWindow();
        if (active != null && active.isVisible()) {
            windows.add(active);
        }
        for (Window window : Window.getWindows()) {
            if (window.isVisible() && !windows.contains(window)) {
                windows.add(window);
            }
        }
        return windows;
    }

    private static Window activeWindow() {
        Window active = KeyboardFocusManager.getCurrentKeyboardFocusManager().getActiveWindow();
        if (active != null && active.isVisible()) {
            return active;
        }
        for (Window window : Window.getWindows()) {
            if (window.isVisible() && window.isFocused()) {
                return window;
            }
        }
        return active != null && active.isVisible() ? active : null;
    }

    private static JMenuBar menuBarOf(Object value) {
        if (value instanceof JFrame) {
            return ((JFrame) value).getJMenuBar();
        }
        if (value instanceof JDialog) {
            return ((JDialog) value).getJMenuBar();
        }
        if (value instanceof JMenuBar) {
            return (JMenuBar) value;
        }
        return null;
    }

    private static Container contentOf(Window window) {
        if (window instanceof RootPaneContainer) {
            return ((RootPaneContainer) window).getContentPane();
        }
        return null;
    }

    private static void collectMenu(JMenuBar bar, int depth, Window window, int rootIndex, String path,
            List<Object> menus, List<Object> actions) {
        for (int i = 0; i < bar.getMenuCount(); i++) {
            JMenu menu = bar.getMenu(i);
            if (menu == null) {
                continue;
            }
            String childPath = path + "." + i;
            menus.add(menuEntry(menu, window, rootIndex, childPath));
            collectMenuItems(menu, depth + 1, window, rootIndex, childPath, menus, actions);
        }
    }

    private static void collectMenuItems(JMenu menu, int depth, Window window, int rootIndex, String path,
            List<Object> menus, List<Object> actions) {
        for (int i = 0; i < menu.getItemCount(); i++) {
            Component item = menu.getItem(i);
            if (item == null) {
                continue;
            }
            String childPath = path + "." + i;
            if (item instanceof JMenu) {
                menus.add(menuEntry((JMenu) item, window, rootIndex, childPath));
                collectMenuItems((JMenu) item, depth + 1, window, rootIndex, childPath, menus, actions);
            } else {
                actions.add(actionEntry(item, window, rootIndex, childPath));
            }
        }
    }

    private static void collectPopupMenus(Component[] components, Window window, int rootIndex, List<Object> menus,
            List<Object> actions, IdentityHashMap<Object, Object> seen) {
        for (Component component : components) {
            if (component instanceof JComponent) {
                JComponent jcomponent = (JComponent) component;
                if (jcomponent.getComponentPopupMenu() != null && !seen.containsKey(jcomponent.getComponentPopupMenu())) {
                    seen.put(jcomponent.getComponentPopupMenu(), Boolean.TRUE);
                    JPopupMenu popup = jcomponent.getComponentPopupMenu();
                    String path = "popup." + seen.size();
                    menus.add(popupEntry(popup, window, rootIndex, path));
                    collectPopupItems(popup, window, rootIndex, path, menus, actions);
                }
            }
            if (component instanceof Container) {
                collectPopupMenus(((Container) component).getComponents(), window, rootIndex, menus, actions, seen);
            }
        }
    }

    private static void collectPopupItems(JPopupMenu popup, Window window, int rootIndex, String path,
            List<Object> menus, List<Object> actions) {
        for (int i = 0; i < popup.getComponentCount(); i++) {
            Component item = popup.getComponent(i);
            if (item == null) {
                continue;
            }
            String childPath = path + "." + i;
            if (item instanceof JMenu) {
                menus.add(menuEntry((JMenu) item, window, rootIndex, childPath));
                collectMenuItems((JMenu) item, 1, window, rootIndex, childPath, menus, actions);
            } else {
                actions.add(actionEntry(item, window, rootIndex, childPath));
            }
        }
    }

    private static Map<String, Object> popupEntry(JPopupMenu popup, Window window, int rootIndex, String path) {
        Map<String, Object> entry = new LinkedHashMap<String, Object>();
        entry.put("window", titleOf(window));
        entry.put("windowIndex", rootIndex);
        entry.put("path", path);
        entry.put("role", "popupmenu");
        entry.put("text", popup.getName() == null ? "" : popup.getName());
        entry.put("enabled", Boolean.valueOf(popup.isEnabled()));
        return entry;
    }

    private static Map<String, Object> menuEntry(JMenu menu, Window window, int rootIndex, String path) {
        Map<String, Object> entry = new LinkedHashMap<String, Object>();
        entry.put("window", titleOf(window));
        entry.put("windowIndex", rootIndex);
        entry.put("path", path);
        entry.put("role", "menu");
        entry.put("text", cleanText(menu.getText()));
        entry.put("enabled", Boolean.valueOf(menu.isEnabled()));
        return entry;
    }

    private static Map<String, Object> actionEntry(Component component, Window window, int rootIndex, String path) {
        Map<String, Object> entry = new LinkedHashMap<String, Object>();
        entry.put("window", titleOf(window));
        entry.put("windowIndex", rootIndex);
        entry.put("path", path);
        entry.put("role", roleOf(component));
        entry.put("text", labelOf(component));
        entry.put("enabled", Boolean.valueOf(component.isEnabled()));
        return entry;
    }

    /**
     * X-TEA list screens often render one logical row through a custom row
     * object rather than ordinary table cells. Read that object reflectively
     * when it exposes the standard shape (getColumnMap/getCellObjectList and
     * cells with getExternalValue), so the JSON view is not just object names.
     */
    private static Map<String, Object> readCustomTable(JTable table, int maxRows) {
        if (table.getRowCount() == 0 || table.getColumnCount() == 0) {
            return null;
        }
        Object first = table.getValueAt(0, 0);
        if (first == null) {
            return null;
        }
        java.lang.reflect.Method cellListMethod = method(first.getClass(), "getCellObjectList");
        java.lang.reflect.Method columnMapMethod = method(first.getClass(), "getColumnMap");
        if (cellListMethod == null || columnMapMethod == null) {
            return null;
        }
        try {
            Object columnMapObject = columnMapMethod.invoke(first);
            if (!(columnMapObject instanceof Map)) {
                return null;
            }
            Map<?, ?> columnMap = (Map<?, ?>) columnMapObject;
            List<Object> columns = new ArrayList<Object>();
            List<Object> rows = new ArrayList<Object>();
            List<Object> usedColumnKeys = new ArrayList<Object>();
            for (int r = 0; r < table.getRowCount() && r < maxRows; r++) {
                Object rowObject = table.getValueAt(r, 0);
                if (rowObject == null || !cellListMethod.getDeclaringClass().isAssignableFrom(rowObject.getClass())) {
                    return null;
                }
                Object cellListObject = cellListMethod.invoke(rowObject);
                if (!(cellListObject instanceof List)) {
                    return null;
                }
                List<Object> row = new ArrayList<Object>();
                for (Object cell : (List<?>) cellListObject) {
                    Object internalValue = invokeNoArg(cell, "getInternalValue");
                    Object value = invokeNoArg(cell, "getExternalValue");
                    if (value == null) {
                        value = internalValue;
                    }
                    if (r == 0) {
                        columns.add(columnNameFor(columnMap, internalValue, usedColumnKeys));
                    }
                    row.add(brief(value));
                }
                rows.add(row);
            }
            Map<String, Object> value = new LinkedHashMap<String, Object>();
            value.put("columns", columns);
            value.put("rows", rows);
            value.put("rowCount", table.getRowCount());
            value.put("selectedRow", table.getSelectedRow());
            value.put("truncated", table.getRowCount() > maxRows);
            value.put("rowType", first.getClass().getName());
            return value;
        } catch (Exception e) {
            return null;
        }
    }

    private static Object invokeNoArg(Object target, String name) {
        if (target == null) {
            return null;
        }
        java.lang.reflect.Method found = method(target.getClass(), name);
        if (found == null) {
            return null;
        }
        try {
            return found.invoke(target);
        } catch (Exception e) {
            return null;
        }
    }

    private static String columnNameFor(Map<?, ?> columnMap, Object internalValue, List<Object> usedKeys) {
        Object fallback = null;
        for (Map.Entry<?, ?> entry : columnMap.entrySet()) {
            if (usedKeys.contains(entry.getKey())) {
                continue;
            }
            if (entry.getValue() == internalValue) {
                usedKeys.add(entry.getKey());
                return String.valueOf(entry.getKey());
            }
            if (fallback == null && entry.getValue() != null && entry.getValue().equals(internalValue)) {
                fallback = entry.getKey();
            }
        }
        if (fallback != null) {
            usedKeys.add(fallback);
            return String.valueOf(fallback);
        }
        for (Map.Entry<?, ?> entry : columnMap.entrySet()) {
            if (!usedKeys.contains(entry.getKey())) {
                usedKeys.add(entry.getKey());
                return String.valueOf(entry.getKey());
            }
        }
        return "column" + usedKeys.size();
    }

    private static java.lang.reflect.Method method(Class<?> type, String name) {
        if (type == null) {
            return null;
        }
        try {
            java.lang.reflect.Method found = type.getMethod(name);
            if (!found.isAccessible()) {
                found.setAccessible(true);
            }
            return found;
        } catch (NoSuchMethodException e) {
            return null;
        } catch (RuntimeException e) {
            return null;
        }
    }

    private static List<Object> tableRows(JTable table, int maxRows) {
        List<Object> rows = new ArrayList<Object>();
        for (int r = 0; r < table.getRowCount() && r < maxRows; r++) {
            List<Object> row = new ArrayList<Object>();
            for (int c = 0; c < table.getColumnCount() && c < 30; c++) {
                row.add(brief(table.getValueAt(r, c)));
            }
            rows.add(row);
        }
        return rows;
    }

    private static Object readValue(Component component) {
        if (component instanceof JPasswordField) {
            return "********";
        }
        if (component instanceof JTextComponent) {
            return brief(((JTextComponent) component).getText());
        }
        if (component instanceof AbstractButton) {
            return Boolean.valueOf(((AbstractButton) component).isSelected());
        }
        if (component instanceof JComboBox) {
            return brief(((JComboBox<?>) component).getSelectedItem());
        }
        if (component instanceof JTable) {
            JTable table = (JTable) component;
            return "rows=" + table.getRowCount() + ", selected=" + table.getSelectedRow();
        }
        if (component instanceof JTree) {
            JTree tree = (JTree) component;
            return tree.getSelectionPath() == null ? "no-selection" : tree.getSelectionPath().toString();
        }
        if (component instanceof JList) {
            return ((JList<?>) component).getSelectedIndex();
        }
        if (component instanceof JTabbedPane) {
            JTabbedPane tabs = (JTabbedPane) component;
            return tabs.getSelectedIndex() < 0 ? "none" : tabs.getTitleAt(tabs.getSelectedIndex());
        }
        if (component instanceof JSpinner) {
            return brief(((JSpinner) component).getValue());
        }
        if (component instanceof JSlider) {
            return ((JSlider) component).getValue();
        }
        return labelOf(component);
    }

    /**
     * Set a value through X-TEA's editable-field wrapper when the selected
     * component is an inner Swing widget. XFTextField, XFDateField, XFCheckBox
     * and related classes keep their own value state and formatting rules.
     */
    private static boolean setCustomFieldValue(Component target, Object requestedValue) {
        Component parent = target.getParent();
        while (parent != null) {
            String className = parent.getClass().getName();
            if (className.startsWith("xeadDriver.XF")) {
                java.lang.reflect.Method setter = method(parent.getClass(), "setValue");
                if (setter != null && setter.getParameterTypes().length == 1) {
                    try {
                        setter.invoke(parent, requestedValue == null ? "" : String.valueOf(requestedValue));
                        return true;
                    } catch (Exception e) {
                        throw new BridgeException("UI_FAILURE",
                                "custom field setValue failed: " + describe(e));
                    }
                }
            }
            parent = parent.getParent();
        }
        return false;
    }

    private static void selectCombo(JComboBox<?> combo, Object value) {
        if (value instanceof Number) {
            int index = ((Number) value).intValue();
            if (index < 0 || index >= combo.getItemCount()) {
                throw new BridgeException("INVALID_INDEX", "combo index out of range: " + index);
            }
            combo.setSelectedIndex(index);
            return;
        }
        String text = String.valueOf(value);
        for (int i = 0; i < combo.getItemCount(); i++) {
            if (text.equals(String.valueOf(combo.getItemAt(i)))) {
                combo.setSelectedIndex(i);
                return;
            }
        }
        for (int i = 0; i < combo.getItemCount(); i++) {
            if (String.valueOf(combo.getItemAt(i)).contains(text)) {
                combo.setSelectedIndex(i);
                return;
            }
        }
        throw new BridgeException("ITEM_NOT_FOUND", "combo item not found: " + text);
    }

    private static TreePath treePath(JTree tree, Object requested, int row) {
        if (requested instanceof List) {
            TreePath path = new TreePath(tree.getModel().getRoot());
            List<?> indices = (List<?>) requested;
            int start = 0;
            if (!indices.isEmpty() && intValue(indices.get(0), -1) == 0) {
                start = 1;
            }
            for (int i = start; i < indices.size(); i++) {
                int index = intValue(indices.get(i), -1);
                Object parent = path.getLastPathComponent();
                if (index < 0 || index >= tree.getModel().getChildCount(parent)) {
                    throw new BridgeException("INVALID_PATH", "tree child index out of range: " + index);
                }
                path = path.pathByAddingChild(tree.getModel().getChild(parent, index));
            }
            return path;
        }
        if (requested != null) {
            String text = String.valueOf(requested);
            if (text.matches("\\d+(\\.\\d+)*")) {
                TreePath path = new TreePath(tree.getModel().getRoot());
                String[] parts = text.split("\\.");
                int start = parts.length > 1 && "0".equals(parts[0]) ? 1 : 0;
                for (int i = start; i < parts.length; i++) {
                    int index = Integer.parseInt(parts[i]);
                    Object parent = path.getLastPathComponent();
                    if (index < 0 || index >= tree.getModel().getChildCount(parent)) {
                        throw new BridgeException("INVALID_PATH", "tree child index out of range: " + index);
                    }
                    path = path.pathByAddingChild(tree.getModel().getChild(parent, index));
                }
                return path;
            }
            for (int i = 0; i < tree.getRowCount(); i++) {
                TreePath path = tree.getPathForRow(i);
                if (path != null && path.toString().contains(text)) {
                    return path;
                }
            }
            throw new BridgeException("PATH_NOT_FOUND", "tree path not found: " + text);
        }
        if (row < 0 || row >= tree.getRowCount()) {
            throw new BridgeException("INVALID_ROW", "tree row out of range: " + row);
        }
        return tree.getPathForRow(row);
    }

    private static int indexOfTab(JTabbedPane tabs, String title) {
        for (int i = 0; i < tabs.getTabCount(); i++) {
            if (title.equals(tabs.getTitleAt(i))) {
                return i;
            }
        }
        for (int i = 0; i < tabs.getTabCount(); i++) {
            if (tabs.getTitleAt(i).contains(title)) {
                return i;
            }
        }
        return -1;
    }

    private static int columnIndex(JTable table, Object spec) {
        if (spec instanceof Number) {
            return ((Number) spec).intValue();
        }
        if (spec == null) {
            return 0;
        }
        String text = String.valueOf(spec);
        for (int c = 0; c < table.getColumnCount(); c++) {
            if (text.equals(table.getColumnName(c))) {
                return c;
            }
        }
        for (int c = 0; c < table.getColumnCount(); c++) {
            if (table.getColumnName(c).contains(text)) {
                return c;
            }
        }
        try {
            return Integer.parseInt(text);
        } catch (NumberFormatException e) {
            throw new BridgeException("INVALID_COLUMN", "column not found: " + text);
        }
    }

    private static String roleOf(Component component) {
        if (component instanceof JMenu) {
            return "menu";
        }
        if (component instanceof javax.swing.JMenuItem) {
            return "menuitem";
        }
        if (component instanceof JTabbedPane) {
            return "tab";
        }
        if (component instanceof JTable) {
            return "table";
        }
        if (component instanceof JTree) {
            return "tree";
        }
        if (component instanceof JList) {
            return "list";
        }
        if (component instanceof JComboBox) {
            return "combobox";
        }
        if (component instanceof JCheckBox) {
            return "checkbox";
        }
        if (component instanceof JRadioButton) {
            return "radio";
        }
        if (component instanceof JTextComponent) {
            return "text";
        }
        if (component instanceof JSpinner) {
            return "spinner";
        }
        if (component instanceof JSlider) {
            return "slider";
        }
        if (component instanceof AbstractButton) {
            return "button";
        }
        if (component instanceof JLabel) {
            return "label";
        }
        if (component instanceof Window) {
            return "window";
        }
        if (component instanceof JMenuBar) {
            return "menubar";
        }
        if (component instanceof javax.swing.JPopupMenu) {
            return "popupmenu";
        }
        if (component instanceof JTextArea) {
            return "textarea";
        }
        return component.getClass().getSimpleName().toLowerCase();
    }

    private static String normalizeRole(String role) {
        if (role == null) {
            return null;
        }
        String value = role.trim().toLowerCase();
        if (value.equals("menuitem") || value.equals("menu-item") || value.equals("item")) {
            return "menuitem";
        }
        if (value.equals("textfield") || value.equals("textarea") || value.equals("editor")) {
            return "text";
        }
        if (value.equals("dialog") || value.equals("frame")) {
            return "window";
        }
        return value;
    }

    private static String labelOf(Component component) {
        if (component instanceof JPasswordField) {
            return "********";
        }
        if (component instanceof JTextComponent) {
            return brief(((JTextComponent) component).getText());
        }
        if (component instanceof AbstractButton) {
            return cleanText(((AbstractButton) component).getText());
        }
        if (component instanceof JLabel) {
            return cleanText(((JLabel) component).getText());
        }
        if (component instanceof JComboBox) {
            Object selected = ((JComboBox<?>) component).getSelectedItem();
            return selected == null ? "" : brief(selected);
        }
        if (component instanceof Window) {
            return titleOf((Window) component);
        }
        if (component instanceof JTabbedPane) {
            JTabbedPane tabs = (JTabbedPane) component;
            int index = tabs.getSelectedIndex();
            return index < 0 ? "" : tabs.getTitleAt(index);
        }
        String name = component.getName();
        if (name != null) {
            return name;
        }
        javax.accessibility.AccessibleContext context = component.getAccessibleContext();
        if (context != null) {
            String accessibleName = context.getAccessibleName();
            if (accessibleName != null) {
                return brief(accessibleName);
            }
        }
        return "";
    }

    private static String describeComponent(Component component) {
        String text = labelOf(component);
        if (text.length() > 80) {
            text = text.substring(0, 80) + "...";
        }
        return roleOf(component) + "[" + component.getClass().getSimpleName() + "]"
                + (text.isEmpty() ? "" : " '" + text + "'") + "@" + Integer.toHexString(System.identityHashCode(component));
    }

    private static String cleanText(String text) {
        if (text == null) {
            return "";
        }
        String value = text.replaceAll("<[^>]+>", " ").replace("&", "");
        value = value.replaceAll("\\s+", " ").trim();
        return brief(value);
    }

    private static String brief(Object value) {
        if (value == null) {
            return null;
        }
        String text = String.valueOf(value);
        if (text.length() > MAX_TEXT) {
            return text.substring(0, MAX_TEXT) + "...";
        }
        return text;
    }

    private static String titleOf(Window window) {
        if (window == null) {
            return "";
        }
        if (window instanceof Frame) {
            String title = ((Frame) window).getTitle();
            return title == null ? "" : title;
        }
        if (window instanceof Dialog) {
            String title = ((Dialog) window).getTitle();
            return title == null ? "" : title;
        }
        return "";
    }

    private static Map<String, Object> windowSummary(Window window, int index) {
        Map<String, Object> item = new LinkedHashMap<String, Object>();
        item.put("ref", "w" + index);
        item.put("title", titleOf(window));
        item.put("class", window.getClass().getName());
        item.put("modal", window instanceof Dialog ? Boolean.valueOf(((Dialog) window).isModal()) : Boolean.FALSE);
        item.put("active", Boolean.valueOf(window == activeWindow()));
        item.put("visible", Boolean.valueOf(window.isVisible()));
        item.put("bounds", bounds(window));
        return item;
    }

    private static Map<String, Object> bounds(Component component) {
        Rectangle rectangle = component.getBounds();
        Map<String, Object> value = new LinkedHashMap<String, Object>();
        value.put("x", rectangle.x);
        value.put("y", rectangle.y);
        value.put("width", rectangle.width);
        value.put("height", rectangle.height);
        return value;
    }

    private static int modifiers(Object value) {
        int modifiers = 0;
        if (value instanceof List) {
            for (Object item : (List<?>) value) {
                modifiers |= modifier(String.valueOf(item));
            }
        } else if (value != null) {
            for (String item : String.valueOf(value).split("[,+]")) {
                modifiers |= modifier(item.trim());
            }
        }
        return modifiers;
    }

    private static int modifier(String name) {
        if ("ctrl".equalsIgnoreCase(name) || "control".equalsIgnoreCase(name)) {
            return KeyEvent.CTRL_DOWN_MASK;
        }
        if ("shift".equalsIgnoreCase(name)) {
            return KeyEvent.SHIFT_DOWN_MASK;
        }
        if ("alt".equalsIgnoreCase(name)) {
            return KeyEvent.ALT_DOWN_MASK;
        }
        if ("meta".equalsIgnoreCase(name)) {
            return KeyEvent.META_DOWN_MASK;
        }
        return 0;
    }

    /**
     * Execute a Swing key binding before falling back to raw key events.
     *
     * A synthetic KeyEvent does not run JTextField's "notify-field-accept"
     * action, so Enter would otherwise look like a no-op even though a real
     * key press commits the field. Using the component's InputMap/ActionMap
     * reproduces the binding path without a global Robot.
     */
    private static boolean dispatchKeyBinding(Component receiver, int keyCode, int modifiers) {
        if (receiver instanceof JTextField && keyCode == KeyEvent.VK_ENTER && modifiers == 0) {
            ((JTextField) receiver).postActionEvent();
            return true;
        }
        if (!(receiver instanceof JComponent)) {
            return false;
        }
        JComponent component = (JComponent) receiver;
        KeyStroke stroke = KeyStroke.getKeyStroke(keyCode, modifiers);
        InputMap inputMap = component.getInputMap(JComponent.WHEN_FOCUSED);
        Object binding = inputMap == null ? null : inputMap.get(stroke);
        if (binding == null && component.getInputMap(JComponent.WHEN_ANCESTOR_OF_FOCUSED_COMPONENT) != null) {
            binding = component.getInputMap(JComponent.WHEN_ANCESTOR_OF_FOCUSED_COMPONENT).get(stroke);
        }
        if (binding == null) {
            return false;
        }
        Action action = component.getActionMap() == null ? null : component.getActionMap().get(binding);
        if (action == null) {
            return false;
        }
        action.actionPerformed(new ActionEvent(component, ActionEvent.ACTION_PERFORMED,
                String.valueOf(keyCode), System.currentTimeMillis(), modifiers));
        return true;
    }

    private static int keyCode(String name) {
        String fieldName = name.toUpperCase();
        if (!fieldName.startsWith("VK_")) {
            fieldName = "VK_" + fieldName;
        }
        try {
            Field field = KeyEvent.class.getField(fieldName);
            return field.getInt(null);
        } catch (Exception e) {
            throw new BridgeException("INVALID_KEY", "unknown key: " + name);
        }
    }

    private static Pattern optionalPattern(Object value) {
        if (value == null) {
            return null;
        }
        String text = String.valueOf(value);
        if (text.isEmpty()) {
            return null;
        }
        return pattern(text);
    }

    private static Pattern pattern(String value) {
        try {
            return Pattern.compile(value, Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);
        } catch (PatternSyntaxException e) {
            return Pattern.compile(Pattern.quote(value), Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);
        }
    }

    private static boolean matches(String pattern, String value) {
        return pattern(pattern).matcher(value == null ? "" : value).find();
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> mapValue(Object value) {
        return value instanceof Map ? (Map<String, Object>) value : null;
    }

    private static String stringValue(Object value) {
        return value == null ? null : String.valueOf(value);
    }

    private static int intValue(Object value, int fallback) {
        if (value instanceof Number) {
            return ((Number) value).intValue();
        }
        if (value != null) {
            try {
                return Integer.parseInt(String.valueOf(value));
            } catch (NumberFormatException e) {
                return fallback;
            }
        }
        return fallback;
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
        if (value != null) {
            String text = String.valueOf(value);
            if ("true".equalsIgnoreCase(text) || "yes".equalsIgnoreCase(text) || "1".equals(text)) {
                return true;
            }
            if ("false".equalsIgnoreCase(text) || "no".equalsIgnoreCase(text) || "0".equals(text)) {
                return false;
            }
        }
        return fallback;
    }

    private static String describe(Throwable throwable) {
        if (throwable == null) {
            return "unknown";
        }
        String message = throwable.getMessage();
        return throwable.getClass().getSimpleName() + (message == null ? "" : ": " + message);
    }

    static final class BridgeException extends RuntimeException {
        private static final long serialVersionUID = 1L;
        final String code;

        BridgeException(String code, String message) {
            super(message);
            this.code = code;
        }
    }
}
