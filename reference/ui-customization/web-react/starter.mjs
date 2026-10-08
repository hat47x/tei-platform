// React is provided by the host. No renderer, backend model or business template
// is coupled to this project-customizable *UI implementation* starter.
export const starterMetadata = Object.freeze({
  id:'tei.ui.web.react.action-form.reference', version:'0.1.0',
  target:'web', framework:'react', frameworkMajor:18,
  slots:['header', 'aside', 'footer'],
});
const fieldTypes = new Set(['text','password','number','datetime-local']);
function assertFields(fields) {
  if (!Array.isArray(fields) || fields.length===0) throw Error('invalid_ui_fields');
  const ids=new Set();
  for (const field of fields) {
    if (!field || typeof field.name!=='string' || !/^[A-Za-z][\w]*$/.test(field.name) ||
        ids.has(field.name) || !fieldTypes.has(field.type) || typeof field.label!=='string' ||
        !field.label.trim()) throw Error('invalid_ui_field');
    ids.add(field.name);
  }
}
export function createActionFormStarter(React, {fields, invoke, slots = {}, title = 'Action'} = {}) {
  if (!React || typeof React.createElement !== 'function' || typeof React.useState !== 'function' ||
      typeof invoke !== 'function' || typeof title!=='string') throw Error('invalid_ui_host');
  assertFields(fields);
  for (const key of Object.keys(slots)) {
    if (!starterMetadata.slots.includes(key) || typeof slots[key]!=='function') throw Error('unsupported_slot');
  }
  const h=React.createElement;
  // Uncontrolled HTML fields keep credentials out of persistent React state.
  // The UI validates only transport shape, not domain authorization/validation.
  function ActionForm() {
    const [busy,setBusy]=React.useState(false);
    const [receipt,setReceipt]=React.useState(null);
    async function submit(event) {
      event.preventDefault();
      if (busy) return;
      setBusy(true);setReceipt(null);
      const form=event.currentTarget;
      try {
        const fd=new FormData(form);
        const values=Object.fromEntries(fields.map(f=>[f.name,fd.get(f.name)]));
        if (Object.hasOwn(values,'row')) {
          if (!/^(0|[1-9]\d*)$/.test(String(values.row))) throw Error('invalid_row');
          values.row=Number(values.row);
        }
        const result=await invoke(values);
        setReceipt({kind:result.kind,stage:result.stage,
          command:result.command?.capability??null,
          observation:result.observation?.readIdentity??null});
      } catch {
        setReceipt({kind:'input_or_transport_error',stage:'unavailable'});
      } finally {
        // Avoid retaining a password in the DOM after request completion.
        const password=form.elements?.namedItem?.('credential');
        if (password && 'value' in password) password.value='';
        setBusy(false);
      }
    }
    const stateText={completed:'Operation completed (Observation is separate)',
      business_denied:'Operation not allowed',
      validation_not_satisfied:'Validation not satisfied',
      execution_error:'Operation failed during execution',
      input_or_transport_error:'Input or connection error'};
    const ctx={React,receipt,busy};
    const custom=(name)=>slots[name]?.(ctx)??null;
    const controls=fields.map(f=>h('label',{key:f.name},f.label,
      h('input',{name:f.name,type:f.type,required:f.required!==false,
        autoComplete:f.type==='password'?'off':undefined,
        min:f.type==='number'?0:undefined,defaultValue:f.defaultValue??undefined})));
    return h('main',{'data-tei-ui-plugin':starterMetadata.id},
      custom('header')??h('h1',null,title),
      h('form',{onSubmit:submit,autoComplete:'off'},
        ...controls,h('button',{type:'submit',disabled:busy},busy?'Processing…':'Submit')),
      h('section',{role:'status','aria-live':'polite'},receipt?
        h('p',null,stateText[receipt.kind]??'Unrecognized result'):null),
      custom('aside'),custom('footer'));
  }
  return ActionForm;
}
