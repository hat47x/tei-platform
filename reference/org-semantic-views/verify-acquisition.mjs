/**
 * Synthetic Organization Domain case for Declared -> Acquired -> Disclosed
 * and distinct provider routes/cohorts. NO physical DB/API/Host calls.
 * Run: node --test reference/org-semantic-views/verify-acquisition.mjs
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { fixture, scenario, membershipVersionsAt } from './source-fixture.mjs';

const mapping = Object.freeze({
  data:'Membership', anchor:'D1', fields:['id','person','dept'],
  grain:'Relation occurrence', requestedScope:'all-for-selected-department',
  sourceRevision:fixture.sourceRevision,
});
// Provider route choices are observation/scenario data, not semantic Person/Relation facts.
const routes = Object.fromEntries(scenario.syntheticProviderRoutes.map(r => [r.id, r]));
function read(routeName, options={}, catalog=routes) {
  const route = catalog[routeName];
  if(!route) return {status:'unsupported',reason:'unresolved-route'};
  if(options.requestedScope==='all-world')
    return {status:'unsupported',reason:'undeclared-scope'};
  if(mapping.sourceRevision!==fixture.sourceRevision)
    return {status:'unsupported',reason:'source-version'};
  const physicalFields=options.physicalFields||mapping.fields;
  const physicalPaths=options.physicalPaths||['Membership'];
  const rows=membershipVersionsAt(fixture,route.validAt,route.knownAt)
    .filter(m=>m.dept===mapping.anchor);
  // These statements deliberately model access, not merely the projected response.
  const acquisition={route:routeName,provider:route.provider,mapping:route.mapping,
    snapshot:route.snapshot,validAt:route.validAt,knownAt:route.knownAt,
    fields:[...physicalFields],paths:[...physicalPaths],rowIds:rows.map(x=>x.id),
    traceCompleteness:options.traceCompleteness??'complete'};
  if(physicalPaths.includes('Occupancy'))
    acquisition.occupancyRowsInspected=fixture.occupancy.map(x=>x.id);
  const missing=mapping.fields.filter(f=>!physicalFields.includes(f));
  if(missing.length)
    return {status:'unsupported',reason:'physical-fields-missing',acquisition};
  const resultRows=rows.map(row=>Object.fromEntries(mapping.fields.map(f=>[f,row[f]])));
  return {status:'ok',mapping,acquisition,
    slice:{rows:resultRows,complete:options.complete??'unknown',
      selectedAnchor:mapping.anchor,sourceRevision:fixture.sourceRevision,
      grain:mapping.grain,epoch:options.epoch??7,provider:route.provider}};
}
function disclose(readResult,consumer='table') {
  if(readResult.status!=='ok')return {status:'unsupported'};
  if(consumer!=='table' && consumer!=='graph')return {status:'unsupported'};
  const {rows}=readResult.slice;
  return {status:'ok',consumer,
    rows:rows.map(r=>consumer==='table'
      ?{edgeId:r.id,person:r.person,department:r.dept}
      :{relationId:r.id,from:r.person,to:r.dept}),
    evidence:{sourceRevision:readResult.slice.sourceRevision,
      anchor:readResult.slice.selectedAnchor,complete:readResult.slice.complete,
      grain:readResult.slice.grain,provider:readResult.slice.provider}};
}
const sortedBag=(rows)=>rows.map(r=>[r.id,r.person,r.dept].join('|')).sort();
function compareRoutes(a,b,cohortWitness=null) {
  if(a.status!=='ok'||b.status!=='ok')return 'not_established';
  if(!cohortWitness || cohortWitness.kind!=='independent-fixture-cohort' ||
      cohortWitness.appliesTo?.length!==2 ||
      ![a.acquisition.route,b.acquisition.route].every(r=>cohortWitness.appliesTo.includes(r)) ||
      a.slice.complete!==true||b.slice.complete!==true ||
      a.slice.selectedAnchor!==b.slice.selectedAnchor ||
      a.slice.grain!==b.slice.grain) return 'not_established';
  return JSON.stringify(sortedBag(a.slice.rows))===JSON.stringify(sortedBag(b.slice.rows))
    ?'fixture_consistent':'contradicted';
}
function physicalFootprint(readResult) {
  if(readResult.status!=='ok')return {status:'unresolved'};
  const a=readResult.acquisition;
  if(a.traceCompleteness!=='complete')return {status:'unverified',reason:'unknown-physical-access'};
  const extraFields=a.fields.filter(f=>!mapping.fields.includes(f));
  const extraPaths=a.paths.filter(p=>p!==mapping.data);
  return {extraFields,extraPaths,
    status:extraFields.length||extraPaths.length?'requires-explanation':'declared-only'};
}
const independentCohort={kind:'independent-fixture-cohort',appliesTo:['replica','primary']};

test('baseline scenario and source revision remain separately bound',()=>{
  assert.equal(fixture.sourceRevision,scenario.sourceRevisionRef);
  assert.equal(routes.replica.validAt,scenario.baseline.validAt);
  assert.equal(routes.replica.knownAt,scenario.baseline.knownAt);
});
test('same Semantic Membership kind has two distinct physical route identities',()=>{
  assert.equal(mapping.data,'Membership');
  assert.notEqual(routes.replica.provider,routes.primary.provider);
  assert.notEqual(routes.replica.mapping,routes.primary.mapping);
});
test('logical D1 mapping is unaffected by availability of D2 and Occupancy',()=>{
  const r=read('replica',{complete:true});
  assert.deepEqual(r.slice.rows.map(x=>x.id),['A1','A2']);
  assert.deepEqual(r.acquisition.paths,['Membership']);
});
test('same source, different known time gives different D1 membership cohorts',()=>{
  assert.deepEqual(read('replica').slice.rows.map(x=>x.id),['A1','A2']);
  assert.deepEqual(read('primary').slice.rows.map(x=>x.id),['A1']);
});
test('same snapshot token s7 from different routes is not a shared cohort proof',()=>{
  const a=read('replica',{complete:true}),b=read('primary',{complete:true});
  assert.equal(a.acquisition.snapshot,b.acquisition.snapshot);
  assert.equal(compareRoutes(a,b),'not_established');
});
test('independent asserted cohort exposes mismatch rather than hiding a relation',()=>{
  const a=read('replica',{complete:true}),b=read('primary',{complete:true});
  assert.equal(compareRoutes(a,b,independentCohort),'contradicted');
});
test('an incomplete replica cannot prove cross-route equality even with witness',()=>{
  const a=read('replica',{complete:false}),b=read('primary',{complete:true});
  assert.equal(compareRoutes(a,b,independentCohort),'not_established');
});
test('one route read twice with equivalent data still needs an independent cohort witness',()=>{
  const a=read('replica',{complete:true}),b=read('replica',{complete:true});
  assert.equal(compareRoutes(a,b),'not_established');
});
test('declared read footprint matches narrow physical acquisition',()=>{
  const r=read('replica',{physicalFields:['id','person','dept'],complete:true});
  assert.deepEqual(physicalFootprint(r),
    {extraFields:[],extraPaths:[],status:'declared-only'});
});
test('consumer output can match despite overfetch of a real source field',()=>{
  const narrow=read('replica',{physicalFields:['id','person','dept'],complete:true});
  const broad=read('replica',{physicalFields:['id','person','dept','provenance'],complete:true});
  assert.deepEqual(disclose(narrow),disclose(broad));
  assert.deepEqual(physicalFootprint(broad),
    {extraFields:['provenance'],extraPaths:[],status:'requires-explanation'});
});
test('provider can read an unmapped Occupancy relation while output is unchanged',()=>{
  const narrow=read('replica',{complete:true});
  const broad=read('replica',{physicalPaths:['Membership','Occupancy'],complete:true});
  assert.deepEqual(disclose(narrow),disclose(broad));
  assert.deepEqual(broad.acquisition.occupancyRowsInspected,['O1','O2']);
  assert.deepEqual(physicalFootprint(broad).extraPaths,['Occupancy']);
});
test('disclosure and acquisition evidence are distinct, not interchangeable',()=>{
  const broad=read('replica',{physicalFields:['id','person','dept','provenance'],complete:true});
  assert.equal('acquisition' in disclose(broad),false);
  assert.equal(broad.acquisition.fields.includes('provenance'),true);
});
test('a missing physically acquired requested field is unsupported',()=>{
  const r=read('replica',{physicalFields:['id','dept'],complete:true});
  assert.equal(r.status,'unsupported');
  assert.equal(r.reason,'physical-fields-missing');
});
test('unknown provider route cannot be inferred by alias name or field equality',()=>{
  assert.deepEqual(read('archive'),{status:'unsupported',reason:'unresolved-route'});
});
test('an undeclared wider requested scope cannot silently expand the Function',()=>{
  assert.equal(read('replica',{requestedScope:'all-world'}).status,'unsupported');
});
test('table and graph disclosures agree on relation identity under same slice',()=>{
  const r=read('replica',{complete:true});
  assert.deepEqual(disclose(r,'table').rows.map(x=>x.edgeId),
    disclose(r,'graph').rows.map(x=>x.relationId));
});
test('same visible row does not choose a write Primary from a read Replica',()=>{
  const r=read('replica',{complete:true});
  assert.equal(r.acquisition.route,'replica');
  assert.equal('writeTarget' in r.slice,false);
  assert.equal('grant' in r.slice,false);
});
test('same source revision, different Provider cohorts cannot claim same current time',()=>{
  const a=read('replica'),b=read('primary');
  assert.equal(a.slice.sourceRevision,b.slice.sourceRevision);
  assert.notEqual(a.acquisition.knownAt,b.acquisition.knownAt);
});
test('exposing the footprint need not expose the actual data or grant read authority',()=>{
  const r=read('replica',{physicalPaths:['Membership','Occupancy']});
  const report=physicalFootprint(r);
  assert.equal('rows' in report,false);
  assert.equal('authority' in report,false);
});
test('local read-only helper does not mutate accepted Source membership versions',()=>{
  const initial=JSON.stringify(fixture.membershipVersions);
  read('replica');read('primary');
  assert.equal(JSON.stringify(fixture.membershipVersions),initial);
});

test('route alias renaming preserves a stable route ID and selected rows',()=>{
  const catalog={...routes,replica:{...routes.replica,alias:'renamed_alias'}};
  assert.notEqual(catalog.replica.alias,routes.replica.alias);
  assert.deepEqual(read('replica',{complete:true},catalog).slice.rows,read('replica',{complete:true}).slice.rows);
});
test('absence of an acquisition trace cannot prove narrow read even when output matches',()=>{
  const observed=read('replica',{traceCompleteness:'unknown',complete:true});
  const narrow=read('replica',{complete:true});
  assert.deepEqual(disclose(observed),disclose(narrow));
  assert.deepEqual(physicalFootprint(observed),
    {status:'unverified',reason:'unknown-physical-access'});
});
