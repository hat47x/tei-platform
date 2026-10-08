// Reference-only use of TEI First Web Target's existing /scenario/submit contract.
// UI fields describe a layout; business identity/validation still belong to TEI.
import {createActionFormStarter} from './starter.mjs';
import {createReferenceSubmitAction} from './action-port.mjs';
export const referenceUIFields = Object.freeze([
  {name:'loginId',label:'Login ID',type:'text'},
  {name:'credential',label:'Credential',type:'password'},
  {name:'from',label:'Browse from',type:'text',defaultValue:'2026-09-18T08:30:00Z'},
  {name:'row',label:'Selected row',type:'number',defaultValue:0},
]);
export function createReferenceOrderEntryUI(React,{fetchImpl,slots}={}) {
  return createActionFormStarter(React,{
    title:'Submit selected order',fields:referenceUIFields,
    invoke:createReferenceSubmitAction({fetchImpl}),slots
  });
}
