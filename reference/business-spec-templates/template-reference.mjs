// Reference-only business specification template proposal and upgrade checks.
// This module never writes TEI Accepted sources or creates Authorization grants.
export class TemplateError extends Error {
  constructor(code){super(code);this.code=code;this.name='TemplateError';}
}
function fail(code){throw new TemplateError(code);}
function valid(s){return typeof s==='string'&&s.trim()!=='';}
function uniq(a){return new Set(a).size===a.length;}
export function inspectTemplate(template){
  if(!template||template.schemaVersion!=='tei-business-spec-template-reference-v0'||
    template.artifactRole!=='business-specification-template-candidate'||template.canonical!==false||
    !valid(template.templateId)||!valid(template.templateVersion)||
    !Array.isArray(template.sourceBindings)||!Array.isArray(template.projectDecisions))fail('invalid_template');
  if(['framework','renderer','css','component','layout'].some(k=>k in template))fail('ui_layer_leak');
  const views=new Set();const keys=[];
  for(const b of template.sourceBindings){
    if(!b||!['data','function','business'].includes(b.view)||!valid(b.source)||
      !Array.isArray(b.refs)||!b.refs.length||!b.refs.every(valid)||!uniq(b.refs))fail('invalid_source_binding');
    views.add(b.view);for(const id of b.refs)keys.push(`${b.view}\u0000${b.source}\u0000${id}`);
  }
  if(!['data','function','business'].every(k=>views.has(k))||!uniq(keys))fail('invalid_view_coverage');
  const questions=template.projectDecisions;
  if(!questions.length||!questions.every(x=>x&&valid(x.id)&&valid(x.question))||
     !uniq(questions.map(x=>x.id)))fail('invalid_project_questions');
  return {templateId:template.templateId,templateVersion:template.templateVersion,
    refCount:keys.length,viewCoverage:[...views],questionCount:questions.length};
}
export function proposeAdoption(template, {projectId, decisions={}}={}){
  inspectTemplate(template);
  if(!valid(projectId)||!decisions||typeof decisions!=='object'||Array.isArray(decisions))fail('invalid_project');
  const required=new Set(template.projectDecisions.map(x=>x.id));
  for(const key of Object.keys(decisions)){if(!required.has(key))fail('unrecognized_project_decision');}
  const answered=[],unresolved=[];
  for(const q of template.projectDecisions){
    const choice=decisions[q.id];
    if(choice && choice.status==='human-confirmed'&&valid(choice.answer)&&valid(choice.sourceRef))
      answered.push({id:q.id,answer:choice.answer,sourceRef:choice.sourceRef});
    else unresolved.push(q.id);
  }
  // Even a complete, confirmed set of project decisions does not commit the
  // candidate as TEI Accepted meaning: a separate authority owns that action.
  return {artifactRole:'project-business-proposal',canonical:false,accepted:false,
    projectId,template:{id:template.templateId,version:template.templateVersion},
    sourceRefs:template.sourceBindings.map(b=>({...b,refs:[...b.refs]})),answered,unresolved,
    next:'explicit_shared_acceptance_required'};
}
export function inspectUpgrade(oldVersion,newVersion,projectProposal){
  inspectTemplate(oldVersion);inspectTemplate(newVersion);
  if(oldVersion.templateId!==newVersion.templateId||
    !projectProposal||projectProposal.template?.id!==oldVersion.templateId||
    projectProposal.template.version!==oldVersion.templateVersion)fail('upgrade_scope_mismatch');
  const signature=x=>JSON.stringify({sources:x.sourceBindings,decisions:x.projectDecisions});
  const changed=signature(oldVersion)!==signature(newVersion);
  return {templateId:oldVersion.templateId,from:oldVersion.templateVersion,to:newVersion.templateVersion,
    changed,requiresReview:changed,acceptedProjectChanged:false,
    next:changed?'explicit_upgrade_review_required':'no_semantic_change_detected'};
}
