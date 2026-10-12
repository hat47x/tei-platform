import {previewLayoutMove, inspectViewLayout, LAYOUT_SCHEMA, EDIT_SCHEMA} from './dfd-view-layout.mjs';

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
const state = {dfd:null, dfdHash:null, layout:null, layoutHash:null, world:null, pending:null, selected:null};
let selectionGeneration=0;
function clearSelection(){
  selectionGeneration++;
  state.selected=null;
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
  const minX=Math.min(0,...positions.map(p=>p.x-24));
  const minY=Math.min(0,...positions.map(p=>p.y-24));
  const maxX=Math.max(1100,...positions.map(p=>p.x+230));
  const maxY=Math.max(600,...positions.map(p=>p.y+100));
  const width=maxX-minX,height=maxY-minY;
  diagram.setAttribute('viewBox',`${minX} ${minY} ${width} ${height}`);
  diagram.style.width=width+'px';
  diagram.style.height=height+'px';
  all.forEach((node,i)=>{
    const pos=positions[i];
    centers.set(key(node.kind,node.id),{x:pos.x+95,y:pos.y+28});
  });
  const edges=[];
  for(const flow of state.dfd.flows||[])edges.push({from:key('step',flow.from),to:key('step',flow.to),kind:'control'});
  for(const flow of state.dfd.information?.informationFlows||[])edges.push({from:key(flow.from.kind,flow.from.id),to:key(flow.to.kind,flow.to.id),kind:'information'});
  for(const t of state.dfd.triggers||[])edges.push({from:key('trigger',t.id),to:key('step',t.targetStep),kind:'control'});
  for(const assignment of state.dfd.information?.responsibilities||[])
    edges.push({from:key('business-role',assignment.role),to:key('step',assignment.step),kind:'responsibility'});
  for(const step of state.dfd.steps)
    for(const ref of step.functionRefs||[])
      edges.push({from:key('step',step.id),to:key('function',ref.id),kind:'function'});
  for(const edge of edges){
    const a=centers.get(edge.from),b=centers.get(edge.to);if(!a||!b)continue;
    const line=svg('line',{x1:a.x,y1:a.y,x2:b.x,y2:b.y,class:'edge '+edge.kind});
    diagram.append(line);
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
async function selectNode(node){
  const generation=++selectionGeneration;
  state.selected=key(node.kind,node.id);
  const d=state.dfd;
  const step=node.kind==='step'?d.steps.find(s=>s.id===node.id):null;
  const candidate={kind:node.kind,id:node.id,authority:'reference-only',
    source:step?.source||null,functionRefs:step?.functionRefs||[],explicitMeaning:step?.explicitMeaning||[],
    handoffs:step?(d.handoffs||[]).filter(h=>h.from?.procedure===d.procedureId&&h.from.step===node.id)
      .map(h=>({to:h.to,condition:h.condition,authority:h.authority,runtimeBinding:'not-inferred'})):[]};
  $('details').replaceChildren();
  const pre=document.createElement('pre');pre.textContent=JSON.stringify(candidate,null,2);$('details').append(pre);
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
  state.layout=state.pending.proposedLayout;
  state.layoutHash=await digest(new TextEncoder().encode(JSON.stringify(state.layout)));
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
  const file=ev.target.files[0];if(!file)return;
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
    state.dfd=d;state.dfdHash=hash;state.layout=initialLayout;state.layoutHash=initialHash;
    state.pending=null;dragging=null;clearSelection();
    $('preview').hidden=true;$('apply').disabled=true;
    render();status('投影を読み込みました。選択と移動は参照・表示用です。');
  }catch(e){state.dfd=null;state.layout=null;state.pending=null;dragging=null;clearSelection();
    $('apply').disabled=true;$('preview').hidden=true;render();status('DFD投影を読み込めません: '+e.message,true)}
});
$('layout-file').addEventListener('change',async ev=>{
  if(!state.dfd)return status('先にDFD投影を選択してください。',true);
  const file=ev.target.files[0];if(!file)return;
  try{
    const bytes=await file.arrayBuffer();const layout=JSON.parse(new TextDecoder().decode(bytes));
    const hash=await digest(bytes);
    // Inspect without synthesizing any edit or changing the layout.
    inspectViewLayout(state.dfd,state.dfdHash,layout,hash);
    state.layout=layout;state.layoutHash=hash;state.pending=null;dragging=null;
    $('apply').disabled=true;$('preview').hidden=true;render();status('配置ファイルを照合して読み込みました。');
  }catch(e){status('配置ファイルを拒否しました: '+e.message,true)}
});
$('world-file').addEventListener('change',async ev=>{
  try{state.world=await jsonFile(ev.target.files[0]);status('Worldを読み込みました。版情報の真正性は未確認です。');}
  catch(e){state.world=null;status('Worldを読み込めません: '+e.message,true)}
});
