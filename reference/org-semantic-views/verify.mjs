/**
 * Independent read-only synthetic oracle for TEI organization semantic views.
 * No TEI import, adapter, actual authorization, side effect, or provider.
 * Run with: node reference/org-semantic-views/verify.mjs
 * This script checks fixture arithmetic and candidate interpretation, not production correctness.
 */
import { fixture, baselineMemberships } from './source-fixture.mjs';
function runFixture() {
  const checks=[];
  function eq(name,actual,expected) {
    const equal=JSON.stringify(actual)===JSON.stringify(expected);
    checks.push({name,pass:equal,actual,expected});
    if (!equal) throw new Error(`${name}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
  const people=fixture.persons;
  const memberships=baselineMemberships(fixture).map(({ id, person, dept })=>({ id, person, dept }));
  const allocations=fixture.allocations;
  const projects=fixture.projects;
  const position=fixture.position;
  const occupancy=fixture.occupancy;
  const distinct=arr=>[...new Set(arr)].sort();
  const membershipsFor=dept=>memberships.filter(x=>x.dept===dept);
  const fteBasisPoints=dept=>membershipsFor(dept).reduce((acc,m)=>acc+allocations.find(a=>a.relation===m.id).basisPoints,0);
  const activeOn=(date)=>occupancy.filter(o=>o.position===position.id&&o.from<=date&&(o.to===null||date<o.to)).map(o=>o.person);
  eq("person-identity-count",people.length,3);
  eq("membership-occurrence-count",memberships.length,4);
  eq("distinct-person-across-departments",distinct(memberships.map(m=>m.person)),["P1","P2","P3"]);
  eq("department-1-members",membershipsFor("D1").map(x=>x.person),["P1","P2"]);
  eq("department-2-members",membershipsFor("D2").map(x=>x.person),["P2","P3"]);
  eq("distinct-not-sum-of-group-counts",distinct(memberships.map(x=>x.person)).length,3);
  eq("fte-D1-as-declared",fteBasisPoints("D1"),160);
  eq("fte-D2-as-declared",fteBasisPoints("D2"),140);
  eq("fte-total-as-declared",fteBasisPoints("D1")+fteBasisPoints("D2"),300);
  const d1join=membershipsFor("D1").flatMap(m=>projects.map(p=>({member:m.id,project:p.id,person:m.person})));
  eq("join-row-occurrences",d1join.length,4);
  eq("join-distinct-memberships",distinct(d1join.map(x=>x.member)),["A1","A2"]);
  eq("join-does-not-create-people",distinct(d1join.map(x=>x.person)),["P1","P2"]);
  eq("join-naive-fte-is-multiplied",d1join.reduce((s,x)=>s+allocations.find(a=>a.relation===x.member).basisPoints,0),320);
  eq("effective-time-before-boundary",activeOn("2027-06-30"),["P2"]);
  eq("effective-time-at-boundary",activeOn("2027-07-01"),["P3"]);
  eq("occupancy-does-not-imply-direct-membership",memberships.some(m=>m.person==="P3"&&m.dept===position.dept),false);
  eq("two-relations-same-person-remain-distinct",memberships.filter(m=>m.person==="P2").map(m=>m.id),["A2","A3"]);
  const direct={id:"A3",source:"accepted-direct"};
  const derived={person:"P3",dept:"D1",source:"rule-derived",bridgeRuleAccepted:false};
  eq("direct-and-derived-provenance-not-equivalent",direct.source===derived.source,false);
  function interpret(change,selectionEpoch=7) {
    if(change.epoch!==selectionEpoch) return {kind:"stale"};
    if(change.targetKind==="aggregate") return {kind:"unsupported",reason:"noninvertible-grain"};
    if(change.targetKind==="derived-relation") return {kind:"unresolved",reason:"no-unique-source-edge"};
    if(change.targetKind==="person"&&people.includes(change.id)) return {kind:"candidate",targetKind:"Person",targetId:change.id,action:"select"};
    if(change.targetKind==="membership"&&memberships.some(m=>m.id===change.id)) {
      return {kind:"candidate",targetKind:"Membership",targetId:change.id,action:"unlink"};
    }
    return {kind:"unsupported",reason:"unknown-target"};
  }
  eq("node-selection-target",interpret({epoch:7,targetKind:"person",id:"P2"}),{kind:"candidate",targetKind:"Person",targetId:"P2",action:"select"});
  eq("edge-unlink-target",interpret({epoch:7,targetKind:"membership",id:"A3"}),{kind:"candidate",targetKind:"Membership",targetId:"A3",action:"unlink"});
  eq("edge-not-person-deletion",interpret({epoch:7,targetKind:"membership",id:"A3"}).targetKind==="Person",false);
  eq("aggregate-edit-rejected",interpret({epoch:7,targetKind:"aggregate",id:"count-people"}),{kind:"unsupported",reason:"noninvertible-grain"});
  eq("derived-relation-edit-unresolved",interpret({epoch:7,targetKind:"derived-relation",id:"P3-D1"}),{kind:"unresolved",reason:"no-unique-source-edge"});
  eq("stale-selection-not-actionable",interpret({epoch:6,targetKind:"membership",id:"A3"}),{kind:"stale"});
  eq("unknown-displayed-target-not-actionable",interpret({epoch:7,targetKind:"membership",id:"ROW3"}),{kind:"unsupported",reason:"unknown-target"});
  const justD1=membershipsFor("D1");
  eq("partial-dept-display-not-global-membership",justD1.length===memberships.length,false);
  eq("source-unchanged-by-readonly-interpreter",memberships.map(m=>m.id),["A1","A2","A3","A4"]);
  return {reference:"org-semantic-view-synthetic",status:"PASS",checks:checks.length,checked:checks.map(c=>c.name),note:"Only isolated deterministic JavaScript calculations. No TEI Query, Function Context, real provider, GUI, database or Host.",fixture:{personCount:3,membershipCount:4,fteBasisPoints:300}};
}
const summary = runFixture();
console.log(JSON.stringify(summary, null, 2));
