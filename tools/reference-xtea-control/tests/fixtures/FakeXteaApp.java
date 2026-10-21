import java.awt.BorderLayout;
import java.awt.Dimension;
import java.awt.GridLayout;
import java.awt.event.ActionEvent;
import java.awt.event.ActionListener;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import javax.swing.JButton;
import javax.swing.JCheckBox;
import javax.swing.JComboBox;
import javax.swing.JDialog;
import javax.swing.JFrame;
import javax.swing.JLabel;
import javax.swing.JMenu;
import javax.swing.JMenuBar;
import javax.swing.JMenuItem;
import javax.swing.JPanel;
import javax.swing.JScrollPane;
import javax.swing.JTabbedPane;
import javax.swing.JTable;
import javax.swing.JTextField;
import javax.swing.SwingUtilities;
import javax.swing.table.DefaultTableModel;

/** Small Swing fixture used to exercise the control protocol without X-TEA. */
public final class FakeXteaApp {
    private final JFrame frame = new JFrame("Fake X-TEA Control Fixture");
    private final JTextField input = new JTextField("initial", 24);
    private final JLabel status = new JLabel("idle");
    private final DefaultTableModel tableModel = new DefaultTableModel(
            new Object[][] { { "A", "Alpha" }, { "B", "Beta" } }, new String[] { "Code", "Name" });
    private final JTable table = new JTable(tableModel);
    private final JCheckBox enabled = new JCheckBox("Enabled", true);
    private final JComboBox<String> mode = new JComboBox<String>(new String[] { "Fast", "Safe" });

    private FakeXteaApp() {
        input.setName("input");
        input.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                status.setText("enter:" + input.getText());
            }
        });
        status.setName("status");
        table.setName("table");
        enabled.setName("enabled");
        mode.setName("mode");

        JMenuBar bar = new JMenuBar();
        JMenu file = new JMenu("File");
        JMenuItem open = new JMenuItem("Open");
        open.setName("open");
        open.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                status.setText("opened");
            }
        });
        JMenuItem exit = new JMenuItem("Exit");
        exit.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                frame.dispose();
            }
        });
        file.add(open);
        file.add(exit);
        JMenu edit = new JMenu("Edit");
        JMenuItem copy = new JMenuItem("Copy");
        copy.setName("copy");
        copy.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                input.setText(input.getText() + "-copied");
            }
        });
        edit.add(copy);
        bar.add(file);
        bar.add(edit);
        frame.setJMenuBar(bar);

        JPanel top = new JPanel(new GridLayout(0, 2, 8, 8));
        top.add(new JLabel("Input"));
        top.add(input);
        top.add(new JLabel("Mode"));
        top.add(mode);
        top.add(enabled);
        top.add(new JLabel(""));

        JButton run = new JButton("Run");
        run.setName("run");
        run.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                status.setText("ran:" + input.getText());
            }
        });
        JButton dialogButton = new JButton("Open Dialog");
        dialogButton.setName("openDialog");
        dialogButton.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                showDialog();
            }
        });
        JPanel actions = new JPanel();
        actions.add(run);
        actions.add(dialogButton);
        actions.add(status);

        JTabbedPane tabs = new JTabbedPane();
        JPanel main = new JPanel(new BorderLayout(8, 8));
        main.add(top, BorderLayout.NORTH);
        main.add(new JScrollPane(table), BorderLayout.CENTER);
        main.add(actions, BorderLayout.SOUTH);
        tabs.addTab("Main", main);
        tabs.addTab("Detail", new JLabel("detail"));

        List<CustomRow> customRows = new ArrayList<CustomRow>();
        customRows.add(CustomRow.of("A", "Alpha"));
        customRows.add(CustomRow.of("B", "Beta"));
        JTable customTable = new JTable(new CustomTableModel(customRows));
        customTable.setName("customTable");
        tabs.addTab("Custom", new JScrollPane(customTable));
        tabs.setName("tabs");

        frame.setDefaultCloseOperation(JFrame.DISPOSE_ON_CLOSE);
        frame.setContentPane(tabs);
        frame.setPreferredSize(new Dimension(720, 420));
        frame.pack();
        frame.setLocationRelativeTo(null);
        frame.setVisible(true);
    }

    private void showDialog() {
        final JDialog dialog = new JDialog(frame, "Fake Dialog", true);
        JTextField dialogInput = new JTextField("dialog-value", 20);
        dialogInput.setName("dialogInput");
        JButton ok = new JButton("OK");
        ok.setName("dialogOk");
        ok.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                status.setText("dialog:" + input.getText());
                dialog.dispose();
            }
        });
        JPanel panel = new JPanel(new GridLayout(0, 2, 8, 8));
        panel.add(new JLabel("Value"));
        panel.add(dialogInput);
        panel.add(new JLabel(""));
        panel.add(ok);
        dialog.setContentPane(panel);
        dialog.pack();
        dialog.setLocationRelativeTo(frame);
        dialog.setVisible(true);
    }

    /** Mimics X-TEA's one-cell-per-row table whose value is a row object. */
    public static final class CustomRow {
        private final Map<String, Object> columnMap = new LinkedHashMap<String, Object>();
        private final List<CustomCell> cells = new ArrayList<CustomCell>();

        static CustomRow of(String code, String name) {
            CustomRow row = new CustomRow();
            row.add("Code", code);
            row.add("Name", name);
            return row;
        }

        private void add(String column, String value) {
            columnMap.put(column, value);
            cells.add(new CustomCell(value));
        }

        public Map<String, Object> getColumnMap() {
            return columnMap;
        }

        public List<CustomCell> getCellObjectList() {
            return cells;
        }
    }

    public static final class CustomCell {
        private final Object value;

        CustomCell(Object value) {
            this.value = value;
        }

        public Object getInternalValue() {
            return value;
        }

        public Object getExternalValue() {
            return value;
        }
    }

    private static final class CustomTableModel extends javax.swing.table.AbstractTableModel {
        private static final long serialVersionUID = 1L;
        private final List<CustomRow> rows;

        CustomTableModel(List<CustomRow> rows) {
            this.rows = rows;
        }

        @Override
        public int getRowCount() {
            return rows.size();
        }

        @Override
        public int getColumnCount() {
            return 1;
        }

        @Override
        public String getColumnName(int column) {
            return "row";
        }

        @Override
        public Object getValueAt(int rowIndex, int columnIndex) {
            return rows.get(rowIndex);
        }

        @Override
        public boolean isCellEditable(int rowIndex, int columnIndex) {
            return false;
        }
    }

    public static void main(String[] args) {
        SwingUtilities.invokeLater(new Runnable() {
            @Override
            public void run() {
                new FakeXteaApp();
            }
        });
    }
}
