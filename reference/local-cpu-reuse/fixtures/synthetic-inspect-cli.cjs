// Synthetic tei.cli.inspect/v0 emitter used ONLY by read-only subprocess transport tests.
const {appendFileSync}=require('node:fs');
const [command,wflag,workspace,kflag,kind,iflag,id,...extra]=process.argv.slice(2);
if(command!=='inspect'||wflag!=='-workspace'||kflag!=='-subject-kind'||
   iflag!=='-subject-id'||extra.length||workspace!==process.env.TEI_FIXTURE_WORKSPACE) {
  process.exitCode=2;
} else {
  appendFileSync(process.env.TEI_FIXTURE_LOG,kind+':'+id+'\n');
  const subject={kind,id};
  let canonical;
  switch(kind) {
    case 'application':canonical={id,patterns:[
      {id:'order_entry.order_lines',kind:'header-detail',relation:'sales.order.lines'},
      {id:'order_entry.customer_reference',kind:'reference',relation:'sales.order.customer'}]};break;
    case 'application-pattern':canonical=id.endsWith('order_lines')?
      {id,kind:'header-detail',relation:'sales.order.lines'}:
      {id,kind:'reference',relation:'sales.order.customer',
        contractQuery:{contract:'master_data.customer',query:'find_by_id',inputs:[]}};break;
    case 'semantic-relation':canonical={id,target:'sales.order_line',cardinality:'many',keys:[]};break;
    case 'semantic-data':canonical={id,attributes:[],relations:[],candidateKeys:[],functionalDependencies:[]};break;
    default:process.exitCode=2;
  }
  if(canonical) {
    const result={schemaVersion:'tei.operations.inspect/v0',effect:'read-only',subject,
      canonical,worldFingerprint:process.env.TEI_FIXTURE_DRIFT==='1'&&kind==='semantic-relation'?'world:drift':'world:w',
      sourceIndexFingerprint:'index:i',sourceStatus:'indexed',sources:{subject,
        artifacts:[{path:'internal/only.tei',fingerprint:'sha256:fixture'}],
        references:[{subject,artifactPath:'internal/only.tei',role:'definition'}]}};
    process.stdout.write(JSON.stringify({schemaVersion:'tei.cli.inspect/v0',ok:true,result}));
  }
}
