package xeadDriver;

import javax.swing.JPanel;
import javax.swing.JTextField;

/** Test double with the same wrapper/setValue shape as X-TEA fields. */
public final class FakeWrappedField extends JPanel {
    private static final long serialVersionUID = 1L;
    private final JTextField field = new JTextField("initial", 16);
    private Object value = "initial";

    public FakeWrappedField() {
        add(field);
    }

    public JTextField getInnerField() {
        return field;
    }

    public void setValue(Object value) {
        this.value = value == null ? "" : String.valueOf(value);
        field.setText(String.valueOf(this.value));
    }

    public Object getValue() {
        return value;
    }
}
