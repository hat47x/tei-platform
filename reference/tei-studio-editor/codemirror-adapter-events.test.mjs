import test from "node:test";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";

// Loads the real editor.ts + codemirror.ts + codemirror-coordinates.ts.
// ONLY the CodeMirror external packages are faked: this checks dispatch/listener
// wiring but is not an actual CodeMirror DOM/keyboard test.
const mocks = {
  "@codemirror/state": [
    'export class EditorState {',
    '  static readOnly={of:value=>({kind:"readOnly",value})};',
    '  static create({doc,extensions}) {',
    '    return {doc:{toString:()=>doc,length:doc.length},',
    '      selection:{main:{from:0,to:0}},extensions};',
    '  }',
    '}'
  ].join("\n"),
  "@codemirror/view": [
    'export const keymap={of:bindings=>({kind:"keymap",bindings})};',
    'export class EditorView {',
    '  static editable={of:value=>({kind:"editable",value})};',
    '  static lineWrapping={kind:"wrap"};',
    '  static contentAttributes={of:value=>({kind:"contentAttributes",value})};',
    '  static updateListener={of:fn=>({kind:"updateListener",fn})};',
    '  constructor({state}) {',
    '    this.state=state; this.focused=false; this.destroyed=false;',
    '    this.diagnostics=[]; globalThis.__fakeCMViews.push(this);',
    '  }',
    '  dispatch(spec) {',
    '    if(this.destroyed) throw Error("DISPATCH_AFTER_DESTROY");',
    '    const docChanged=Boolean(spec.changes), selectionSet=Boolean(spec.selection);',
    '    if(docChanged) {',
    '      const text=spec.changes.insert;',
    '      this.state.doc={toString:()=>text,length:text.length};',
    '    }',
    '    if(selectionSet) {',
    '      const {anchor,head}=spec.selection;',
    '      if(anchor<0||head<0||anchor>this.state.doc.length||head>this.state.doc.length)',
    '        throw Error("Selection points outside of document");',
    '      this.state.selection.main={from:Math.min(anchor,head),to:Math.max(anchor,head)};',
    '    }',
    '    if(spec.kind==="diagnostics") this.diagnostics=spec.diagnostics;',
    '    for(const ext of this.state.extensions) if(ext.kind==="updateListener") {',
    '      ext.fn({state:this.state,selectionSet,docChanged});',
    '    }',
    '  }',
    '  focus(){this.focused=true;}',
    '  destroy(){this.destroyed=true;}',
    '}'
  ].join("\n"),
  "@codemirror/lint": [
    'export const lintGutter=()=>({kind:"lintGutter"});',
    'export const setDiagnostics=(_state,diagnostics)=>({kind:"diagnostics",diagnostics});'
  ].join("\n"),
  "@codemirror/commands": 'export const standardKeymap=[{key:"End",run:()=>true,shift:()=>true}];'
};

globalThis.__fakeCMViews=[];
registerHooks({
  resolve(specifier,context,nextResolve) {
    if(Object.hasOwn(mocks,specifier)) return {
      url:"data:text/javascript,"+encodeURIComponent(mocks[specifier]),
      shortCircuit:true
    };
    return nextResolve(specifier,context);
  }
});
const {createReadOnlyTextEditor}=await import("./editor.ts");
const {codeMirrorRenderer}=await import("./codemirror.ts");
const view=()=>globalThis.__fakeCMViews.at(-1);
const doc=(text="a😀x\r\nsecond🧪!",version="v1")=>({
  id:"source",version,text,
  diagnostics:[{
    id:"span",message:"second",severity:"warning",
    range:{start:{line:2,column:1},end:{line:2,column:7},unit:"utf16-unit"}
  }]
});
const span=(a,b,unit="utf16-unit")=>({
  start:{line:2,column:a},end:{line:2,column:b},unit
});

test("actual adapter under CM stubs: Host 6..15 -> view 5..14 -> notification 6..15",()=>{
  const events=[];
  const editor=createReadOnlyTextEditor({},codeMirrorRenderer,doc(),e=>events.push(e));
  const cm=view();
  assert.equal(cm.state.doc.toString(),"a😀x\nsecond🧪!");
  assert.deepEqual(cm.diagnostics,[{
    from:5,to:11,message:"second",severity:"warning"
  }]);
  assert.deepEqual(
    cm.state.extensions.filter(x=>x.kind==="readOnly"||x.kind==="editable"),
    [{kind:"readOnly",value:true},{kind:"editable",value:false}]
  );
  assert.deepEqual(cm.state.extensions.find(x=>x.kind==="contentAttributes").value,
    { "aria-label": "Read-only source text", tabindex: "0" });
  const keyboardBindings=cm.state.extensions.find(x=>x.kind==="keymap").bindings;
  assert.ok(keyboardBindings.some(binding=>binding.key==="End"&&typeof binding.shift==="function"),
    "the standard End/Shift+End selection binding must be installed");
  editor.select(span(1,10));
  assert.deepEqual(cm.state.selection.main,{from:5,to:14});
  assert.deepEqual(editor.snapshot().selection,{from:6,to:15});
  assert.equal(events.length,0,"programmatic select must not notify");
  editor.focus();
  assert.equal(cm.focused,true);
  // Emulates a user-originated CodeMirror update(selectionSet=true).
  // This is NOT a browser keyboard event.
  cm.dispatch({selection:{anchor:5,head:14}});
  assert.deepEqual(events,[{
    documentId:"source",version:"v1",selection:{from:6,to:15}
  }]);
  editor.destroy();
  assert.equal(cm.destroyed,true);
});

test("actual adapter under CM stubs: astral byte columns and CRLF->LF re-map",()=>{
  const events=[];
  const editor=createReadOnlyTextEditor({},codeMirrorRenderer,doc(),e=>events.push(e));
  const cm=view();
  editor.select(span(7,11,"utf8-byte"));
  assert.deepEqual(cm.state.selection.main,{from:11,to:13});
  cm.dispatch({selection:{anchor:11,head:13}});
  assert.deepEqual(events.at(-1).selection,{from:12,to:14});
  editor.setDocument(doc("a😀x\nsecond🧪!","v2"));
  assert.equal(cm.state.doc.toString(),"a😀x\nsecond🧪!");
  cm.dispatch({selection:{anchor:5,head:11}});
  assert.deepEqual(events.at(-1),{
    documentId:"source",version:"v2",selection:{from:5,to:11}
  });
  assert.throws(()=>editor.setDiagnostics("source","v1",[]),
    /STALE_EDITOR_DIAGNOSTICS/);
  editor.destroy();
});
