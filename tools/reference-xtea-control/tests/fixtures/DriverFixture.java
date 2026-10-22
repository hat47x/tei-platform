import java.awt.GridLayout;
import java.io.FileReader;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.util.HashMap;
import javax.swing.JButton;
import javax.swing.JFrame;
import javax.swing.JLabel;
import javax.swing.SwingUtilities;
import org.h2.tools.RunScript;
import org.w3c.dom.Element;
import org.w3c.dom.NodeList;
import javax.xml.parsers.DocumentBuilderFactory;
import xeadDriver.*;

/** External Driver classes, original synthetic model, disposable H2 only. */
public final class DriverFixture {
    public static void main(String[] args) throws Exception {
        String url = "jdbc:h2:" + args[0] + "/db";
        try (Connection db = DriverManager.getConnection(url, "sa", "")) {
            RunScript.execute(db, new FileReader(args[0] + "/init.sql"));
        }
        final Session session = new Session(args[0] + "/fixture.xeaf", "00000", "password", "fixture") {
            // Producers run unchanged; record completion instead of opening an external viewer.
            @Override public void browseFile(java.net.URI uri) {
                try {
                    String name = java.nio.file.Paths.get(uri).getFileName().toString();
                    java.nio.file.Path pending = java.nio.file.Paths.get(args[0], "output-ready.tmp");
                    java.nio.file.Files.write(pending,
                            name.getBytes(java.nio.charset.StandardCharsets.UTF_8));
                    // Readers use existence as completion: publish only after all bytes are written.
                    java.nio.file.Files.move(pending, java.nio.file.Paths.get(args[0], "output-ready"),
                            java.nio.file.StandardCopyOption.ATOMIC_MOVE,
                            java.nio.file.StandardCopyOption.REPLACE_EXISTING);
                } catch (java.io.IOException error) {
                    throw new RuntimeException(error);
                }
            }
        };
        final NodeList functions = DocumentBuilderFactory.newInstance().newDocumentBuilder()
                .parse(args[0] + "/fixture.xeaf").getElementsByTagName("Function");
        // The non-Application Session constructor omits this ordinary UI initializer.
        // Instantiate the original calendar class; do not replace its input/selection behavior.
        java.lang.reflect.Constructor<?> calendarConstructor = Class.forName("xeadDriver.XFCalendar")
                .getDeclaredConstructor(Session.class);
        calendarConstructor.setAccessible(true);
        java.lang.reflect.Field calendarField = Session.class.getDeclaredField("xfCalendar");
        calendarField.setAccessible(true);
        SwingUtilities.invokeAndWait(() -> {
            try {
                calendarField.set(session, calendarConstructor.newInstance(session));
            } catch (ReflectiveOperationException error) {
                throw new RuntimeException(error);
            }
        });
        SwingUtilities.invokeLater(() -> {
            JFrame frame = new JFrame("Driver fixture launcher");
            frame.setLayout(new GridLayout(0, 1));
            JLabel result = new JLabel("ready");
            result.setName("result");
            for (int i = 0; i < functions.getLength(); i++) {
                final Element function = (Element) functions.item(i);
                for (String mode : function.getAttribute("Type").equals("XF200")
                        ? new String[] {"DISPLAY", "ADD", "EDIT"} : new String[] {"DISPLAY"}) {
                    JButton open = new JButton(function.getAttribute("ID") + " " + mode);
                    open.addActionListener(event -> {
                        try {
                            HashMap<String, Object> parameters = new HashMap<>();
                            if (!mode.equals("ADD")) parameters.put("ID", "001");
                            if (mode.equals("EDIT")) parameters.put("INSTANCE_MODE", mode);
                            result.setText("running");
                            result.setText(session.executeFunction(function.getAttribute("ID"), parameters).toString());
                        } catch (Exception error) {
                            result.setText("ERROR " + error);
                            error.printStackTrace();
                        }
                    });
                    frame.add(open);
                }
            }
            JButton observe = new JButton("Observe DB");
            observe.addActionListener(event -> {
                try (Connection db = DriverManager.getConnection(url, "sa", "");
                     ResultSet rows = db.createStatement().executeQuery("SELECT ID,NAME,UPDCOUNTER FROM ITEM ORDER BY ID")) {
                    StringBuilder text = new StringBuilder();
                    while (rows.next()) text.append(rows.getString(1)).append(':')
                            .append(rows.getString(2)).append(':').append(rows.getInt(3)).append(';');
                    try (ResultSet lines = db.createStatement().executeQuery("SELECT ID,SEQ,VALUE,NOTE FROM LINE ORDER BY ID,SEQ")) {
                        while (lines.next()) text.append("L:").append(lines.getString(1)).append(':')
                                .append(lines.getInt(2)).append(':').append(lines.getString(3)).append(';')
                                .append("N:").append(lines.getInt(2)).append(':').append(lines.getString(4)).append(';');
                    }
                    result.setText(text.toString());
                } catch (Exception error) {
                    result.setText("ERROR " + error);
                }
            });
            frame.add(observe);
            frame.add(result);
            frame.pack();
            frame.setVisible(true);
            try {
                java.nio.file.Files.write(java.nio.file.Paths.get(args[0], "ready"), new byte[] {1});
            } catch (java.io.IOException error) {
                throw new RuntimeException(error);
            }
        });
    }
}
