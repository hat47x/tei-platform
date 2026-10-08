// Project-owned UI-only customization: adds a side panel without modifying the
// UI starter or re-declaring the TEI business rule/permission/submit action.
export function projectSidePanel({React,receipt}) {
  return React.createElement('aside',{'aria-label':'Project details panel'},
    React.createElement('h2',null,'Details'),
    React.createElement('p',null,receipt?.command?
      'Result for action: '+receipt.command : 'Select a row and submit to see the result.'));
}
