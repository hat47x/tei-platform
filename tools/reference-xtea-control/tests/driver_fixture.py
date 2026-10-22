"""Create original synthetic input for the external, unmodified X-TEA Driver."""
from pathlib import Path
import xml.etree.ElementTree as ET


def write_fixture(folder: Path) -> Path:
    system = ET.Element("System", Name="Control fixture", Version="1", DateFormat="en00",
                        DatabaseName="jdbc:h2:" + (folder / "db").as_posix(),
                        DatabaseUser="sa", DatabasePassword="", UserTable="USERS",
                        UserVariantsTable="USER_VARIANTS", VariantsTable="VARIANTS",
                        NumberingTable="NUMBERS", SessionTable="SESSIONS",
                        SessionDetailTable="SESSION_DETAIL", CalendarTable="CALENDAR",
                        OutputFolder=str(folder), SystemFont="Dialog")

    def table(name, fields, keys):
        element = ET.SubElement(system, "Table", ID=name, Name=name)
        for i, (field, kind, size) in enumerate(fields):
            ET.SubElement(element, "Field", ID=field, Name=field, Type=kind, Size=str(size),
                          Decimal="0", Nullable="F", Order=str(i), TypeOptions="")
        ET.SubElement(element, "Key", Type="PK", Fields=keys)
        return element

    table("USERS", [("IDUSER", "CHAR", 5), ("TXPASSWORD", "VARCHAR", 100),
                    ("TXNAME", "VARCHAR", 30), ("NREMPLOYEE", "VARCHAR", 10),
                    ("TXEMAIL", "VARCHAR", 30), ("TXMENUS", "VARCHAR", 20),
                    ("DTVALID", "DATE", 10), ("DTEXPIRE", "DATE", 10)], "IDUSER")
    table("SESSIONS", [("TXIPADDRESS", "VARCHAR", 100)], "")
    for name in ("USER_VARIANTS", "VARIANTS", "NUMBERS", "SESSION_DETAIL", "CALENDAR"):
        ET.SubElement(system, "Table", ID=name, Name=name)
    table("ITEM", [("ID", "CHAR", 3), ("NAME", "CHAR", 30)], "ID")
    table("LINE", [("ID", "CHAR", 3), ("SEQ", "INTEGER", 3),
                   ("VALUE", "CHAR", 30), ("NOTE", "VARCHAR", 100)], "ID;SEQ")

    def button(parent, position, number, action, caption):
        ET.SubElement(parent, "Button", Position=str(position), Number=str(number),
                      Action=action, Caption=caption)

    def fields(parent, tag, sources):
        for i, source in enumerate(sources):
            ET.SubElement(parent, tag, DataSource=source, FieldOptions="", Order=str(i))

    for typ in ("100", "110", "200"):
        f = ET.SubElement(system, "Function", ID="F" + typ, Type="XF" + typ,
                          Name="Control XF" + typ, PrimaryTable="ITEM", Size="AUTO",
                          InitialListing="T", OrderBy="ITEM.ID")
        fields(f, "Field" if typ == "200" else "Column", ["ITEM.ID", "ITEM.NAME"])
        button(f, 0, 3, "EXIT", "Close")
        button(f, 5, 9, "OUTPUT", "Excel")
        if typ == "100":
            f.set("DetailFunction", "F200")
            ET.SubElement(f, "Filter", DataSource="ITEM.NAME", FieldOptions="SCAN", Order="0")
            button(f, 1, 6, "ADD", "Add")
        if typ == "110":
            button(f, 1, 6, "NEXT", "Next")
            button(f, 2, 8, "UPDATE", "Save")
        if typ == "200":
            f.set("OrderBy", "")
            button(f, 1, 6, "EDIT", "Save")
            button(f, 2, 8, "DELETE", "Delete")
            button(f, 3, 5, "COPY", "Copy")

    for typ in ("300", "310"):
        f = ET.SubElement(system, "Function", ID="F" + typ, Type="XF" + typ,
                          Name="Control XF" + typ, HeaderTable="ITEM", Size="AUTO")
        fields(f, "Field", ["ITEM.ID", "ITEM.NAME"])
        button(f, 0, 3, "EXIT", "Close")
        if typ == "300":
            f.set("HeaderFunction", "F200")
            detail = ET.SubElement(f, "Detail", Table="LINE", Caption="Lines", Order="0",
                                   OrderBy="LINE.SEQ", InitialListing="T")
            fields(detail, "Column", ["LINE.SEQ", "LINE.VALUE"])
            button(detail, 0, 3, "EXIT", "Close")
            button(detail, 5, 9, "OUTPUT", "Excel")
        else:
            f.set("DetailTable", "LINE")
            f.set("DetailOrderBy", "LINE.SEQ")
            fields(f, "Column", ["LINE.SEQ", "LINE.VALUE"])
            button(f, 1, 6, "ADD_ROW", "Add row")
            button(f, 2, 7, "REMOVE_ROW", "Remove row")
            button(f, 3, 8, "UPDATE", "Save")
            button(f, 5, 9, "OUTPUT", "Excel")

    f = ET.SubElement(system, "Function", ID="F311", Type="XF310", Name="Control long text",
                      HeaderTable="ITEM", DetailTable="LINE", DetailOrderBy="LINE.SEQ", Size="AUTO")
    fields(f, "Field", ["ITEM.ID", "ITEM.NAME"])
    fields(f, "Column", ["LINE.SEQ", "LINE.NOTE"])
    button(f, 0, 3, "EXIT", "Close")
    button(f, 1, 8, "UPDATE", "Save")

    ET.SubElement(system, "Function", ID="F000", Type="XF000", Name="Control XF000",
                  Script='instance.getReturnMap().put("CONTROL_RESULT", "ran");')
    scripts = {
        "TRANSACTION": '''function readName() {
var q=instance.createTableOperator("SELECT NAME FROM ITEM WHERE ID='001'");
q.next(); return q.getValueOf("NAME").toString().trim(); }
var rows=instance.createTableOperator("UPDATE ITEM SET NAME='Pending', UPDCOUNTER=UPDCOUNTER+1 WHERE ID='001'").execute();
var pending=readName();
var p=Java.type("javax.swing.JOptionPane");
var answer=p.showConfirmDialog(null,"Save pending change?","Script transaction",p.YES_NO_OPTION);
if (answer==p.YES_OPTION) { instance.commit(); } else { instance.rollback(); }
instance.getReturnMap().put("UI_RESULT",rows+":"+pending+":"+readName());''',
        "INPUT": '''var d=instance.getSession().getInputDialog(); d.clear();
d.setFont(new (Java.type("java.awt.Font"))("Dialog",0,14));
var f=d.addField("Value", "ALPHA", 0, "VALUE"); f.setValue("initial");
var r=d.request("Script input");
instance.getReturnMap().put("UI_RESULT", r+":"+d.getValueOfFieldByParmID("VALUE"));''',
        "OPTION": '''var d=instance.getSession().getOptionDialog();
d.addOption("First"); d.addOption("Second");
instance.getReturnMap().put("UI_RESULT", d.request("Script option"));''',
        "CHECK": '''var d=instance.getSession().getCheckListDialog(); d.clear();
d.addRow("A", "Alpha", false); d.addRow("B", "Beta", false);
var r=d.request("Script checklist");
instance.getReturnMap().put("UI_RESULT", r+":"+d.getListSelected());''',
        "MESSAGE": '''var p=Java.type("javax.swing.JOptionPane");
p.showMessageDialog(null,"Fixture message"); instance.getReturnMap().put("UI_RESULT","seen");''',
        "CONFIRM": '''var p=Java.type("javax.swing.JOptionPane");
instance.getReturnMap().put("UI_RESULT",p.showConfirmDialog(null,"Fixture confirm","Script confirm",p.YES_NO_OPTION));''',
        "PROMPT": '''var p=Java.type("javax.swing.JOptionPane");
instance.getReturnMap().put("UI_RESULT",p.showInputDialog(null,"Fixture prompt"));''',
        "CHOICE": '''var p=Java.type("javax.swing.JOptionPane");
var options=Java.to(["First","Second"],"java.lang.Object[]");
instance.getReturnMap().put("UI_RESULT",p.showOptionDialog(null,"Fixture choice","Script choice",p.DEFAULT_OPTION,p.QUESTION_MESSAGE,null,options,options[0]));''',
    }
    for name, script in scripts.items():
        ET.SubElement(system, "Function", ID="S" + name, Type="XF000", Name="Script " + name, Script=script)
    for typ, initial in (("ALPHA", "initial"), ("TEXTAREA", "initial"), ("NUMERIC", "1"),
                         ("ZEROFILL", ""), ("DATE", "2026-10-21"), ("CHECKBOX", "false"), ("LISTBOX", "A")):
        script = '''var d=instance.getSession().getInputDialog(); d.clear();
d.setFont(new (Java.type("java.awt.Font"))("Dialog",0,14));
var f=d.addField("Value", "%s", 0, "VALUE"); f.setSize(5);
''' % typ
        if typ == "LISTBOX":
            script += 'f.addItem("First", "A"); f.addItem("Second", "B");\n'
        script += 'f.setValue("%s"); var r=d.request("Script typed input");\n' % initial
        script += 'instance.getReturnMap().put("UI_RESULT",r+":"+d.getValueOfFieldByParmID("VALUE"));'
        ET.SubElement(system, "Function", ID="STYPE_" + typ, Type="XF000", Name="Input " + typ, Script=script)
    ET.SubElement(system, "PrintFont", ID="FONT", PDFFontName="Helvetica", PDFEncoding="Cp1252")
    for typ in ("290", "390"):
        f = ET.SubElement(system, "Function", ID="F" + typ, Type="XF" + typ,
                          Name="Control XF" + typ, PageSize="A4", Direction="NORMAL",
                          Margins="20;20;20;20", WithPageNumber="T")
        f.set("PrimaryTable" if typ == "290" else "HeaderTable", "ITEM")
        ET.SubElement(f, "Phrase" if typ == "290" else "HeaderPhrase", Order="0", Block="PARAGRAPH", Alignment="LEFT",
                      Value="&DataSource(ITEM.NAME)", FontID="FONT", FontSize="12", FontStyle="NORMAL")
        if typ == "390":
            detail = ET.SubElement(f, "Detail", Table="LINE", Order="0", OrderBy="LINE.SEQ",
                                   Caption="Lines", FontID="FONT", FontSize="10", RowNoWidth="20")
            fields(detail, "Column", ["LINE.SEQ", "LINE.VALUE"])
    path = folder / "fixture.xeaf"
    ET.ElementTree(system).write(path, encoding="utf-8", xml_declaration=True)
    (folder / "init.sql").write_text("""
CREATE TABLE USERS (IDUSER CHAR(5), TXPASSWORD VARCHAR(100), TXNAME VARCHAR(30),
 NREMPLOYEE VARCHAR(10), TXEMAIL VARCHAR(30), TXMENUS VARCHAR(20), DTVALID DATE, DTEXPIRE DATE);
INSERT INTO USERS VALUES ('00000','5f4dcc3b5aa765d61d8327deb882cf99','Fixture','','','ALL','2000-01-01','2099-12-31');
CREATE TABLE USER_VARIANTS (IDUSERKUBUN VARCHAR(30), KBUSERKUBUN VARCHAR(10), TXUSERKUBUN VARCHAR(30), SQLIST INT);
INSERT INTO USER_VARIANTS VALUES ('KBCALENDAR','A','Fixture calendar',0);
CREATE TABLE VARIANTS (IDVARIANT VARCHAR(30), TXVALUE VARCHAR(100), TXTYPE VARCHAR(20));
INSERT INTO VARIANTS VALUES ('LOGIN_PERMITTED','T','STRING');
CREATE TABLE NUMBERS (IDNUMBER VARCHAR(30), NRCURRENT INT, NRNUMDIGIT INT,
 TXPREFIX VARCHAR(5), FGWITHCD CHAR(1), UPDCOUNTER INT);
INSERT INTO NUMBERS VALUES ('NRSESSION',1,8,'','F',0);
CREATE TABLE SESSIONS (NRSESSION VARCHAR(30), IDUSER VARCHAR(5), DTLOGIN TIMESTAMP,
 TXIPADDRESS VARCHAR(100), VLVERSION VARCHAR(30), KBSESSIONSTATUS VARCHAR(5), DTLOGOUT TIMESTAMP);
CREATE TABLE SESSION_DETAIL (NRSESSION VARCHAR(30), SQPROGRAM INT, IDMENU VARCHAR(30),
 IDPROGRAM VARCHAR(30), TXPROGRAM VARCHAR(100), DTSTART TIMESTAMP, DTEND TIMESTAMP,
 KBPROGRAMSTATUS VARCHAR(5), TXERRORLOG CLOB);
CREATE TABLE CALENDAR (KBCALENDAR VARCHAR(10), DTOFF DATE);
CREATE TABLE ITEM (ID CHAR(3) PRIMARY KEY, NAME VARCHAR(30), UPDCOUNTER INT DEFAULT 0);
INSERT INTO ITEM(ID,NAME) VALUES ('001','Alpha'),('002','Beta');
CREATE TABLE LINE (ID CHAR(3), SEQ INT, VALUE VARCHAR(30), NOTE VARCHAR(100) DEFAULT 'Note', UPDCOUNTER INT DEFAULT 0,
 PRIMARY KEY(ID,SEQ));
INSERT INTO LINE(ID,SEQ,VALUE) VALUES ('001',1,'First'),('001',2,'Second');
""", encoding="utf-8")
    return path
