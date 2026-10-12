import {previewLayoutMove, inspectViewLayout, LAYOUT_SCHEMA, EDIT_SCHEMA} from './dfd-view-layout.mjs';
import {previewSelectedDFDEdit, BINDING_SCHEMA} from './dfd-selected-edit-bridge.mjs';

const $ = id => document.getElementById(id);
const NS = 'http://www.w3.org/2000/svg';
const svg = (tag, attributes={}) => {
  const el = document.createElementNS(NS,tag);
  for(const [k,v] of Object.entries(attributes)) el.setAttribute(k,String(v));
  return el;
};
const digest = async bytes => 'sha256:' + [...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))]
  .map(x=>x.toString(16).padStart(2,'0')).join('');
const jsonFile = async file => JSON.parse(await file.text());
const state = {dfd:null, dfdHash:null, layout:null, layoutHash:null, world:null, pending:null, selected:null,
  dfdFile:null, procedureFile:null, reviewFile:null, procedure:null,review:null,
  procedureHash:null,reviewHash:null,semanticPreview:null};
let selectionGeneration=0;
let dfdLoadGeneration=0;
let layoutLoadGeneration=0;
let worldLoadGeneration=0;
const semanticLoadGeneration={procedure:0,review:0};
let semanticTextGeneration=0;
function invalidateSemanticPreview(){
  state.semanticPreview=null;
  $('semantic-preview').hidden=true;
  $('semantic-export').disabled=true;
  $('semantic-pending').textContent='';
}
function populateStepOptions(id,entries){
  const control=$(id);
  const prior=control.value;
  control.replaceChildren();
  for(const entry of entries){
    const option=document.createElement('option');
    option.value=entry.id;
    option.textContent=entry.label||entry.id;
    control.append(option);
  }
  control.value=entries.some(entry=>entry.id===prior)?prior:entries[0]?.id||'';
  return entries.length>0;
}
function updateSemanticControls(){
  const selected=state.selected ? JSON.parse(state.selected) : null;
  const isTrigger=selected?.[0]==='trigger';
  const isStep=selected?.[0]==='step';
  const isInbound=selected?.[0]==='party'||selected?.[0]==='store';
  const isInfoOrigin=isStep||isInbound;
  const ready=Boolean(state.dfd&&state.procedure&&state.review);
  $('trigger-label').disabled=!isTrigger;
  if(isTrigger&&state.dfd){
    const trigger=state.dfd.triggers.find(t=>t.id===selected[1]);
    $('trigger-label').value=trigger?.label||'';
  }else $('trigger-label').value='';
  const actionControl=$('step-action');
  const modes=['assign-business-role','add-control-transition','add-information-flow'];
  if(isInbound)actionControl.value='add-information-flow';
  else if(!modes.includes(actionControl.value))actionControl.value=modes[0];
  actionControl.disabled=!isStep||!ready;
  const mode=actionControl.value;
  const occupied=new Set((state.review?.responsibilities||[])
    .filter(r=>r.step===selected?.[1]).map(r=>r.role));
  const roles=isStep&&ready?(state.review?.businessRoles||[])
    .filter(role=>!occupied.has(role.id)):[];
  const hasRoles=populateStepOptions('step-role',roles);
  $('step-role').disabled=!isStep||!ready||mode!=='assign-business-role'||!hasRoles;
  const steps=isInfoOrigin&&ready?(state.dfd?.steps||[])
    .filter(step=>!isStep||step.id!==selected[1]):[];
  const hasSteps=populateStepOptions('step-transition-to',steps);
  const activeControl=isStep&&ready&&mode==='add-control-transition';
  $('step-transition-to').disabled=!activeControl||!hasSteps;
  $('step-transition-kind').disabled=!activeControl;
  if(!['sequence','branch','exception'].includes($('step-transition-kind').value))
    $('step-transition-kind').value='branch';
  $('step-transition-condition').disabled=!activeControl||
    $('step-transition-kind').value==='sequence';
  const activeInfo=isInfoOrigin&&ready&&mode==='add-information-flow';
  const infoKind=$('step-info-to-kind');
  if(isInbound)infoKind.value='step';
  else if(!['step','party','store'].includes(infoKind.value))infoKind.value='party';
  infoKind.disabled=!activeInfo||isInbound;
  const destinations=infoKind.value==='step'?steps:
    infoKind.value==='party'?(state.dfd?.information?.externalParties||[]):
      (state.dfd?.information?.stores||[]);
  const hasDestination=populateStepOptions('step-info-to-id',activeInfo?destinations:[]);
  $('step-info-to-id').disabled=!activeInfo||!hasDestination;
  $('step-info-id').disabled=!activeInfo;
  $('step-info-payload').disabled=!activeInfo;
  $('semantic-propose').disabled=!ready||!(isTrigger||isStep&&(
    mode==='assign-business-role'&&hasRoles||
    mode==='add-control-transition'&&hasSteps||
    mode==='add-information-flow'&&hasDestination)||
    isInbound&&mode==='add-information-flow'&&hasDestination);
}
function clearSelection(){
  selectionGeneration++;
  state.selected=null;
  invalidateSemanticPreview();
  updateSemanticControls();
  $('details').replaceChildren();
  $('details').textContent='要素を選択すると参照元を表示します。';
}
function status(message,error=false){$('status').textContent=message;$('status').classList.toggle('error',error)}
function nodes() {
  const d = state.dfd;
  if(!d) return [];
  const info=d.information||{};
  const all=[...d.steps.map(s=>({kind:'step',id:s.id,label:s.id+' '+(s.label||''),
    manual:s.noApplicationFunction})),...d.triggers.map(t=>({kind:'trigger',id:t.id,label:t.label||t.id})),
    ...(info.roles||[]).map(x=>({kind:'business-role',id:x.id,label:x.label||x.id})),
    ...(info.externalParties||[]).map(x=>({kind:'party',id:x.id,label:x.label||x.id})),
    ...(info.stores||[]).map(x=>({kind:'store',id:x.id,label:x.label||x.id}))];
  const functions=new Set(d.steps.flatMap(s=>(s.functionRefs||[]).map(f=>f.id)));
  for(const id of functions)all.push({kind:'function',id,label:id});
  return all;
}
const key = (kind,id)=>JSON.stringify([kind,id]);
function positionMap(){return new Map((state.layout?.nodes||[]).map(p=>[key(p.kind,p.id),p]));}
function render() {
  const diagram=$('diagram');diagram.replaceChildren();
  if(!state.dfd){$('export').disabled=true;return;}
  const all=nodes(), assigned=positionMap(), centers=new Map();
  const positions=all.map((node,i)=>
    assigned.get(key(node.kind,node.id))||{x:40+(i%4)*260,y:50+Math.floor(i/4)*115});
  // Preserve readable node size for large Procedures and for nodes moved
  // beyond the initial board. Coordinates are view-only, not canonical.
  let minX=0,minY=0,maxX=1100,maxY=600;
  // A Procedure can have many Steps. Avoid argument-list overflow from
  // Math.min(...positions) and keep the diagram scrollable.
  for(const p of positions){
    minX=Math.min(minX,p.x-24);minY=Math.min(minY,p.y-24);
    maxX=Math.max(maxX,p.x+230);maxY=Math.max(maxY,p.y+100);
  }
  const width=maxX-minX,height=maxY-minY;
  diagram.setAttribute('viewBox',`${minX} ${minY} ${width} ${height}`);
  diagram.style.width=width+'px';
  diagram.style.height=height+'px';
  all.forEach((node,i)=>{
    const pos=positions[i];
    centers.set(key(node.kind,node.id),{x:pos.x+95,y:pos.y+28});
  });
  const defs=svg('defs');
  const marker=svg('marker',{id:'dfd-arrow',markerWidth:8,markerHeight:8,refX:7,refY:4,orient:'auto'});
  marker.append(svg('path',{d:'M0,0 L8,4 L0,8 z',fill:'#64748b'}));
  defs.append(marker);diagram.append(defs);
  const edges=[];
  for(const flow of state.dfd.flows||[])edges.push({from:key('step',flow.from),to:key('step',flow.to),
    kind:flow.kind==='exception'?'exception':flow.kind==='branch'?'branch':'control',
    label:flow.kind==='sequence'?'':flow.kind+': '+flow.condition});
  for(const flow of state.dfd.information?.informationFlows||[])edges.push({from:key(flow.from.kind,flow.from.id),to:key(flow.to.kind,flow.to.id),kind:'information',label:flow.payload});
  for(const t of state.dfd.triggers||[])edges.push({from:key('trigger',t.id),to:key('step',t.targetStep),kind:'control'});
  for(const assignment of state.dfd.information?.responsibilities||[])
    edges.push({from:key('business-role',assignment.role),to:key('step',assignment.step),kind:'responsibility'});
  for(const step of state.dfd.steps)
    for(const ref of step.functionRefs||[])
      edges.push({from:key('step',step.id),to:key('function',ref.id),kind:'function'});
  for(const edge of edges){
    const a=centers.get(edge.from),b=centers.get(edge.to);if(!a||!b)continue;
    const line=svg('line',{x1:a.x,y1:a.y,x2:b.x,y2:b.y,class:'edge '+edge.kind,
      'marker-end':'url(#dfd-arrow)'});
    diagram.append(line);
    if(edge.label){
      const caption=svg('text',{x:Math.round((a.x+b.x)/2),y:Math.round((a.y+b.y)/2)-8,
        class:'edge-label'});
      caption.textContent=edge.label.slice(0,48);
      diagram.append(caption);
    }
  }
  all.forEach((node,i)=>{
    const p=positions[i];
    const g=svg('g',{transform:`translate(${p.x},${p.y})`,class:'node '+(node.manual?'manual':'')+(node.kind==='party'?' external':'')+(state.selected===key(node.kind,node.id)?' selected':'')});
    g.dataset.kind=node.kind;g.dataset.id=node.id;
    g.setAttribute('tabindex','0');
    g.setAttribute('role','button');
    g.setAttribute('aria-label',node.kind+': '+node.label+' (参照のみ)');
    g.append(svg('rect',{width:190,height:56,rx:8}));
    const label=svg('text',{x:10,y:22});label.textContent=(node.label||'').slice(0,27);g.append(label);
    const note=svg('text',{x:10,y:43});note.textContent=node.kind;g.append(note);
    g.addEventListener('pointerdown',ev=>{if(ev.button!==0)return; selectNode(node);dragStart(ev,node);});
    g.addEventListener('keydown',ev=>{
      if(ev.key==='Enter'||ev.key===' '){
        ev.preventDefault();selectNode(node);return;
      }
      const delta={ArrowLeft:[-1,0],ArrowRight:[1,0],ArrowUp:[0,-1],ArrowDown:[0,1]}[ev.key];
      if(!delta)return;
      ev.preventDefault();selectNode(node);
      const step=ev.shiftKey?1:20;
      proposeNodeMove(node,p.x+delta[0]*step,p.y+delta[1]*step);
    });
    diagram.append(g);
  });
  $('export').disabled=!state.layout;
}
// Links are derived exclusively from explicit edges in the displayed projection.
 // Neither the link nor its label asserts a canonical or accepted relationship.
function relatedDiagramNodes(d,node) {
  const all=nodes();
  const existing=new Set(all.map(item=>key(item.kind,item.id)));
  const seen=new Set();
  const related=[];
  function add(kind,id){
    const k=key(kind,id);
    if(k!==key(node.kind,node.id)&&existing.has(k)&&!seen.has(k)){
      seen.add(k);related.push({kind,id});
    }
  }
  const info=d.information||{};
  if(node.kind==='step'){
    const step=d.steps.find(x=>x.id===node.id);
    for(const ref of step?.functionRefs||[])add('function',ref.id);
    for(const r of info.responsibilities||[])if(r.step===node.id)add('business-role',r.role);
    for(const trigger of d.triggers||[])if(trigger.targetStep===node.id)add('trigger',trigger.id);
    for(const flow of d.flows||[]){
      if(flow.from===node.id)add('step',flow.to);
      if(flow.to===node.id)add('step',flow.from);
    }
  }else if(node.kind==='function'){
    for(const step of d.steps)if((step.functionRefs||[]).some(ref=>ref.id===node.id))
      add('step',step.id);
  }else if(node.kind==='business-role'){
    for(const r of info.responsibilities||[])if(r.role===node.id)add('step',r.step);
  }else if(node.kind==='trigger'){
    const trigger=d.triggers.find(t=>t.id===node.id);
    if(trigger)add('step',trigger.targetStep);
  }
  for(const flow of info.informationFlows||[]){
    if(flow.from.kind===node.kind&&flow.from.id===node.id)
      add(flow.to.kind,flow.to.id);
    if(flow.to.kind===node.kind&&flow.to.id===node.id)
      add(flow.from.kind,flow.from.id);
  }
  return related;
}
function diagramReferenceSource(d,node){
  const info=d.information||{};
  const refs={'step':d.steps,'trigger':d.triggers,
    'business-role':info.roles,'party':info.externalParties,'store':info.stores};
  return (refs[node.kind]||[]).find(x=>x.id===node.id)||null;
}
async function selectNode(node){
  const generation=++selectionGeneration;
  state.selected=key(node.kind,node.id);
  // Step selections are distinct review scopes: never carry an endpoint,
  // condition, or flow identifier forward from a previous selection.
  $('step-action').value='assign-business-role';
  $('step-transition-kind').value='branch';
  $('step-transition-condition').value='';
  $('step-info-to-kind').value=node.kind==='party'||node.kind==='store'?'step':'party';
  $('step-info-id').value='';
  $('step-info-payload').value='';
  invalidateSemanticPreview();
  updateSemanticControls();
  const d=state.dfd;
  const item=diagramReferenceSource(d,node);
  const step=node.kind==='step'?item:null;
  const related=relatedDiagramNodes(d,node);
  const candidate={kind:node.kind,id:node.id,authority:'reference-only',
    source:item?.source||null,functionRefs:step?.functionRefs||[],
    explicitMeaning:step?.explicitMeaning||[],
    semanticData:node.kind==='store'?item?.semanticData||null:null,
    relatedDiagramNodes:related,
    handoffs:step?(d.handoffs||[]).filter(h=>h.from?.procedure===d.procedureId&&h.from.step===node.id)
      .map(h=>({to:h.to,condition:h.condition,authority:h.authority,runtimeBinding:'not-inferred'})):[]};
  $('details').replaceChildren();
  const pre=document.createElement('pre');pre.textContent=JSON.stringify(candidate,null,2);$('details').append(pre);
  if(related.length){
    const links=document.createElement('div');
    for(const target of related){
      const button=document.createElement('button');
      button.type='button';
      button.textContent=target.kind+': '+target.id;
      button.addEventListener('click',()=>{
        if(state.dfd!==d||state.selected!==key(node.kind,node.id))return;
        const match=nodes().find(x=>x.kind===target.kind&&x.id===target.id);
        if(match)selectNode(match);
      });
      links.append(button);
    }
    $('details').append(links);
  }
  if(step&&state.world&&$('revision').value){
    const navigation=document.createElement('pre');
    try {const {navigateReferenceBusinessStep}=await import('../studio-authoring/bridge.mjs');navigation.textContent=JSON.stringify(navigateReferenceBusinessStep(state.world,$('revision').value,d,node.id),null,2)}
    catch(err){navigation.textContent='参照先未解決：'+err.message}
    if(state.dfd!==d || selectionGeneration!==generation)return;
    $('details').append(navigation);
  }
  if(state.dfd!==d || selectionGeneration!==generation)return;
  render();
  // Recreating the SVG should not strand keyboard focus on a detached node.
  const focused=[...$('diagram').children].find(el=>el.dataset?.kind===node.kind&&el.dataset?.id===node.id);
  focused?.focus?.({preventScroll:true});
}
let dragging=null;
function dragStart(event,node){
  const rect=$('diagram').getBoundingClientRect();
  const p=positionMap().get(key(node.kind,node.id));
  const index=nodes().findIndex(n=>n.kind===node.kind&&n.id===node.id);
  const position=p||{x:40+(index%4)*260,y:50+Math.floor(index/4)*115};
  dragging={node,startX:event.clientX,startY:event.clientY,position,rect,pointerId:event.pointerId};
  $('diagram').setPointerCapture(event.pointerId);
}
function proposeNodeMove(node,x,y){
  if(!state.dfd||!state.layout)return;
  const edit={schema:EDIT_SCHEMA,id:'move-'+node.kind+'-'+node.id,origin:'studio',
    intent:'図の表示位置だけを調整する',procedureId:state.dfd.procedureId,
    projectionSHA256:state.dfdHash,baseLayoutSHA256:state.layoutHash,
    operation:{kind:'move-node',nodeKind:node.kind,nodeId:node.id,
      x:Math.round(x),y:Math.round(y)}};
  try{
    state.pending=previewLayoutMove(state.dfd,state.dfdHash,state.layout,state.layoutHash,edit);
    $('pending').textContent=JSON.stringify(state.pending.operation,null,2);
    $('preview').hidden=false;$('apply').disabled=false;
    status('配置変更の候補ができました。正本は変更していません。');
  }catch(e){state.pending=null;$('preview').hidden=true;$('apply').disabled=true;status('配置の変更を拒否しました: '+e.message,true)}
}
$('diagram').addEventListener('pointercancel',event=>{
  if(dragging && (event.pointerId===undefined||event.pointerId===dragging.pointerId))dragging=null;
});
$('diagram').addEventListener('pointerup',event=>{
  if(!dragging||!state.dfd||!state.layout)return;
  if(event.pointerId!==undefined && dragging.pointerId!==undefined && event.pointerId!==dragging.pointerId)return;
  const {node,startX,startY,position,rect}=dragging;dragging=null;
  const view=$('diagram').viewBox.baseVal;
  const dx=(event.clientX-startX)*(view.width/rect.width);
  const dy=(event.clientY-startY)*(view.height/rect.height);
  if(Math.abs(dx)+Math.abs(dy)<3)return;
  proposeNodeMove(node,position.x+dx,position.y+dy);
});
$('apply').addEventListener('click',async()=>{
  if(!state.pending)return;
  const candidate=state.pending;
  const layout=candidate.proposedLayout;
  const dfdGeneration=dfdLoadGeneration;
  const generation=layoutLoadGeneration;
  $('apply').disabled=true;
  const hash=await digest(new TextEncoder().encode(JSON.stringify(layout)));
  // A newer DFD, layout selection or move proposal invalidates this hash.
  // Never complete an earlier asynchronous apply against the new view.
  if(dfdGeneration!==dfdLoadGeneration||generation!==layoutLoadGeneration||
     state.pending!==candidate)return;
  state.layout=layout;state.layoutHash=hash;
  state.pending=null;$('preview').hidden=true;$('apply').disabled=true;
  render();status('配置を画面に反映しました。正本には書き込んでいません。');
});
$('export').addEventListener('click',()=>{
  if(!state.layout)return;
  const url=URL.createObjectURL(new Blob([JSON.stringify(state.layout,null,2)+'\n'],{type:'application/json'}));
  const a=document.createElement('a');a.href=url;a.download='business-dfd-layout.json';a.click();
  setTimeout(()=>URL.revokeObjectURL(url),1000);
});
$('dfd-file').addEventListener('change',async ev=>{
  const file=ev.target.files?.[0];
  const generation=++dfdLoadGeneration;
  ++layoutLoadGeneration; // Older layout reads belong to a different DFD.
  // The input selection changed already: no old projection or proposal may
  // remain available while the replacement bytes are still being read.
  state.dfd=null;state.dfdHash=null;state.dfdFile=null;
  state.layout=null;state.layoutHash=null;state.pending=null;dragging=null;
  clearSelection();
  $('preview').hidden=true;$('apply').disabled=true;
  render();
  if(!file){
    status('DFD投影の選択を解除しました。以前の変更候補は無効です。');
    return;
  }
  try{
    const bytes=await file.arrayBuffer();const d=JSON.parse(new TextDecoder().decode(bytes));
    if(d.schemaVersion!=='business-dfd-projection-v0'||d.authority!=='derived-read-only'||d.canonicalProcedureUsed!==false||
      !Array.isArray(d.steps)||!Array.isArray(d.triggers)||!Array.isArray(d.diagnostics)||d.diagnostics.some(x=>x.severity==='error'))
      throw Error('不正または未解決のDFD投影');
    const hash=await digest(bytes);
    const initialLayout={schemaVersion:LAYOUT_SCHEMA,authority:'view-only',procedureId:d.procedureId,projectionSHA256:hash,nodes:[]};
    const initialHash=await digest(new TextEncoder().encode(JSON.stringify(initialLayout)));
    // Do not display a malformed or ambiguous DFD as an editable view.
    inspectViewLayout(d,hash,initialLayout,initialHash);
    if(generation!==dfdLoadGeneration)return;
    state.dfd=d;state.dfdHash=hash;state.dfdFile=file;state.layout=initialLayout;state.layoutHash=initialHash;
    state.pending=null;dragging=null;clearSelection();
    $('preview').hidden=true;$('apply').disabled=true;
    render();status('投影を読み込みました。選択と移動は参照・表示用です。');
  }catch(e){if(generation!==dfdLoadGeneration)return;
    state.dfd=null;state.dfdHash=null;state.dfdFile=null;
    state.layout=null;state.layoutHash=null;state.pending=null;dragging=null;clearSelection();
    $('apply').disabled=true;$('preview').hidden=true;render();status('DFD投影を読み込めません: '+e.message,true)}
});
$('layout-file').addEventListener('change',async ev=>{
  const generation=++layoutLoadGeneration;
  const dfdGeneration=dfdLoadGeneration;
  const file=ev.target.files?.[0];
  // A newly chosen (or cleared) layout revokes the prior view and pending
  // edit before file.arrayBuffer() can yield to other user interactions.
  state.layout=null;state.layoutHash=null;state.pending=null;dragging=null;
  $('apply').disabled=true;$('preview').hidden=true;
  render();
  if(!state.dfd)return status('先にDFD投影を選択してください。',true);
  if(!file)return status('配置ファイルの選択を解除しました。以前の配置と変更候補は無効です。');
  try{
    const bytes=await file.arrayBuffer();const layout=JSON.parse(new TextDecoder().decode(bytes));
    const hash=await digest(bytes);
    // Inspect without synthesizing any edit or changing the layout.
    if(generation!==layoutLoadGeneration||dfdGeneration!==dfdLoadGeneration)return;
    inspectViewLayout(state.dfd,state.dfdHash,layout,hash);
    state.layout=layout;state.layoutHash=hash;state.pending=null;dragging=null;
    $('apply').disabled=true;$('preview').hidden=true;render();status('配置ファイルを照合して読み込みました。');
  }catch(e){if(generation!==layoutLoadGeneration||dfdGeneration!==dfdLoadGeneration)return;
    status('配置ファイルを拒否しました: '+e.message,true)}
});
$('world-file').addEventListener('change',async ev=>{
  const generation=++worldLoadGeneration;
  const file=ev.target.files?.[0];
  // A World snapshot is a navigational hint, not accepted Source. Discard
  // prior navigation before yielding to an asynchronous file read.
  state.world=null;
  clearSelection();
  if(!file){
    status('Worldの選択を解除しました。以前の参照表示は無効です。');
    return;
  }
  try{
    const world=await jsonFile(file);
    if(generation!==worldLoadGeneration)return;
    if(!world||typeof world!=='object'||Array.isArray(world))
      throw Error('INVALID_WORLD_OBJECT');
    state.world=world;
    status('Worldを読み込みました。版情報の真正性は未確認です。');
  }catch(e){
    if(generation!==worldLoadGeneration)return;
    state.world=null;
    status('Worldを読み込めません: '+e.message,true);
  }
});
$('revision').addEventListener('input',()=>{
  // Existing Source details were produced against the previous user-supplied
  // revision label. Require a fresh selection before showing navigation.
  clearSelection();
  status('Worldの版表示を変更しました。参照先を選び直してください（版の真正性は未確認）。');
});

async function readChecked(file,expectedHash,name){
  const bytes=await file.arrayBuffer();
  const hash=await digest(bytes);
  if(hash!==expectedHash)throw Error('DFD_' + name + '_INPUT_CHANGED');
  return hash;
}
async function readReferenceFile(file,expectedVersion){
  const bytes=await file.arrayBuffer();
  const value=JSON.parse(new TextDecoder().decode(bytes));
  if(value.schemaVersion!==expectedVersion)throw Error('UNSUPPORTED_DFD_REFERENCE_SCHEMA');
  return {value,hash:await digest(bytes)};
}
for(const [id,kind,version] of [
  ['procedure-file','procedure','business-procedure-reference-v0'],
  ['review-file','review','business-dfd-review-v0']
]){
  $(id).addEventListener('change',async event=>{
    const file=event.target.files?.[0];
    const generation=++semanticLoadGeneration[kind];
    // Clearing the picker is a new generation too: do not resurrect a file
    // or a proposal through a late asynchronous read.
    invalidateSemanticPreview();
    state[kind]=null;state[kind+'File']=null;state[kind+'Hash']=null;
    updateSemanticControls();
    if(!file){
      status('参照資料の選択を解除しました。以前の変更候補は無効です。');
      return;
    }
    try{
      const {value,hash}=await readReferenceFile(file,version);
      if(generation!==semanticLoadGeneration[kind])return;
      state[kind]=value;state[kind+'File']=file;state[kind+'Hash']=hash;
      updateSemanticControls();
      status('参照資料を読み込みました。正本と同一版であることは未確認です。');
    }catch(err){
      if(generation!==semanticLoadGeneration[kind])return;
      status('参照資料の読み込みを拒否しました: '+err.message,true);
    }
  });
}
$('trigger-label').addEventListener('input',()=>{
  semanticTextGeneration++;
  invalidateSemanticPreview();
});
for(const [id,event] of [
  ['step-role','change'],['step-action','change'],
  ['step-transition-to','change'],['step-transition-kind','change'],
  ['step-transition-condition','input'],['step-info-id','input'],
  ['step-info-to-kind','change'],['step-info-to-id','change'],
  ['step-info-payload','input']
]){
  $(id).addEventListener(event,()=>{
    semanticTextGeneration++;
    invalidateSemanticPreview();
    if(['step-action','step-transition-kind','step-info-to-kind'].includes(id))
      updateSemanticControls();
  });
}
function semanticInputs(){
  return {
    label:$('trigger-label').value,
    role:$('step-role').value,
    action:$('step-action').value,
    transitionTo:$('step-transition-to').value,
    transitionKind:$('step-transition-kind').value,
    transitionCondition:$('step-transition-condition').value,
    infoID:$('step-info-id').value,
    infoToKind:$('step-info-to-kind').value,
    infoToID:$('step-info-to-id').value,
    infoPayload:$('step-info-payload').value
  };
}
function selectedBusinessOperation(selected,input){
  if(selected[0]==='trigger')
    return {kind:'replace-trigger-label',trigger:selected[1],label:input.label};
  // External Parties and conceptual Stores can only propose information
  // entering an explicit business Step, never a direct Store write or RBAC.
  if(selected[0]==='party'||selected[0]==='store'){
    if(input.action!=='add-information-flow'||input.infoToKind!=='step')
      throw Error('DFD_INFORMATION_PROCESS_BOUNDARY_REQUIRED');
    return {kind:'add-information-flow',id:input.infoID,
      from:{kind:selected[0],id:selected[1]},
      to:{kind:'step',id:input.infoToID},payload:input.infoPayload};
  }
  if(selected[0]!=='step')throw Error('UNSUPPORTED_DFD_BROWSER_EDIT_SELECTION');
  switch(input.action){
    case 'assign-business-role':
      return {kind:'assign-business-role',step:selected[1],role:input.role};
    case 'add-control-transition':{
      const operation={kind:'add-control-transition',from:selected[1],
        to:input.transitionTo,transitionKind:input.transitionKind};
      if(input.transitionKind!=='sequence')operation.condition=input.transitionCondition;
      return operation;
    }
    case 'add-information-flow':
      return {kind:'add-information-flow',id:input.infoID,
        from:{kind:'step',id:selected[1]},
        to:{kind:input.infoToKind,id:input.infoToID},payload:input.infoPayload};
    default:throw Error('UNSUPPORTED_DFD_BROWSER_EDIT_MODE');
  }
}
$('semantic-propose').addEventListener('click',async()=>{
  const selected=state.selected?JSON.parse(state.selected):null;
  const dfd=state.dfd,procedure=state.procedure,review=state.review;
  const files=[state.dfdFile,state.procedureFile,state.reviewFile];
  const hashes=[state.dfdHash,state.procedureHash,state.reviewHash];
  const input=semanticInputs();
  const mode=selected?.[0];
  const generation=selectionGeneration;
  const labelGeneration=semanticTextGeneration;
  const inputSnapshot=JSON.stringify(input);
  invalidateSemanticPreview();
  if(!['trigger','step','party','store'].includes(mode)||!files.every(Boolean)||
    $('semantic-propose').disabled||
    (mode==='trigger'&&(!input.label||!input.label.trim()))||
    (mode==='step'&&(
      input.action==='assign-business-role'&&!input.role||
      input.action==='add-control-transition'&&(!input.transitionTo||
        !['sequence','branch','exception'].includes(input.transitionKind)||
        input.transitionKind!=='sequence'&&!input.transitionCondition.trim())||
      input.action==='add-information-flow'&&(!input.infoID.trim()||
        !input.infoToID||!input.infoPayload.trim())))||
    (['party','store'].includes(mode)&&(
      input.action!=='add-information-flow'||input.infoToKind!=='step'||
      !input.infoID.trim()||!input.infoToID||!input.infoPayload.trim()))){
    status('Stepまたは起動契機の選択、参照資料、有効な変更内容が必要です。',true);
    return;
  }
  try{
    for(let i=0;i<files.length;i++)await readChecked(files[i],hashes[i],
      ['PROJECTION','PROCEDURE','REVIEW'][i]);
    if(generation!==selectionGeneration||labelGeneration!==semanticTextGeneration||
       inputSnapshot!==JSON.stringify(semanticInputs())||
       $('semantic-propose').disabled||
       dfd!==state.dfd||procedure!==state.procedure||review!==state.review)
       throw Error('DFD_SELECTION_OR_SOURCE_CHANGED');
    const operation=selectedBusinessOperation(selected,input);
    const binding={schema:BINDING_SCHEMA,id:operation.kind+'-'+selected[1],
      origin:'studio',intent:mode==='trigger'
        ?'起動契機の業務上の説明文だけを変更する'
        :operation.kind==='assign-business-role'
          ?'Stepの担当業務Roleを提案する（認可権限ではない）'
          :operation.kind==='add-control-transition'
            ?'Stepからの業務上の順序・分岐・例外を提案する（自動実行ではない）'
            :'選択した業務要素からの情報の授受を提案する（データ書込みではない）',
      procedureSHA256:state.procedureHash,reviewSHA256:state.reviewHash,
      projectionSHA256:state.dfdHash,
      selection:{kind:mode,id:selected[1]},operation};
    const packet=previewSelectedDFDEdit(dfd,procedure,review,binding);
    if(packet.readyForAcceptance!==false||packet.sharedAcceptanceCandidate!==null||
       packet.sourceWrites.length!==0)throw Error('DFD_UNEXPECTED_ACCEPTANCE_AUTHORITY');
    state.semanticPreview=packet;
    $('semantic-pending').textContent=JSON.stringify({
      selected:packet.selected,change:packet.change,proposedBusinessOperation:operation,
      reviewAuthority:packet.authority,readyForAcceptance:packet.readyForAcceptance,
      sourceWrites:packet.sourceWrites,inputFingerprints:packet.inputFingerprints
    },null,2);
    $('semantic-preview').hidden=false;
    $('semantic-export').disabled=false;
    status('未受理の変更候補を確認できます。正本は変更していません。');
  }catch(err){
    invalidateSemanticPreview();
    status('未受理の変更候補を拒否しました: '+err.message,true);
  }
});
$('semantic-export').addEventListener('click',()=>{
  if(!state.semanticPreview)return;
  const bytes=JSON.stringify(state.semanticPreview,null,2)+'\n';
  const url=URL.createObjectURL(new Blob([bytes],{type:'application/json'}));
  const anchor=document.createElement('a');
  anchor.href=url;anchor.download='business-dfd-unaccepted-review.json';anchor.click();
  setTimeout(()=>URL.revokeObjectURL(url),1000);
});
