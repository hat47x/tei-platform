package main

import (
	"fmt"
)

// The three authored Reference files must be stable while the read-only
// projection is loaded. Workspace revision binding is deliberately NOT
// claimed: that remains the contract of #1120/#1124.
func runGuarded(procedureFile, reviewFile, setFile, workspaceDir string) (projection, []navigationSnapshot, error) {
	paths:=[]string{procedureFile,reviewFile,setFile}
	before:=make([]navigationSnapshot,0,len(paths))
	for _,path:=range paths {
		s,err:=snapshot(path)
		if err!=nil {return projection{},nil,fmt.Errorf("initial source snapshot: %w",err)}
		before=append(before,s)
	}
	p,err:=run(procedureFile,reviewFile,setFile,workspaceDir)
	if err!=nil {return projection{},nil,err}
	for i,path:=range paths {
		after,err:=snapshot(path)
		if err!=nil {return projection{},nil,fmt.Errorf("post-projection source snapshot: %w",err)}
		if after!=before[i] {
			return projection{},nil,fmt.Errorf("DFD_REFERENCE_SOURCE_CHANGED_DURING_PROJECTION: %s",path)
		}
	}
	return p,before,nil
}

func verifySnapshots(qualified []navigationSnapshot) error {
	for _,expected:=range qualified {
		current,err:=snapshot(expected.File)
		if err!=nil {return fmt.Errorf("DFD_REFERENCE_SOURCE_UNREADABLE: %w",err)}
		if current!=expected {
			return fmt.Errorf("DFD_REFERENCE_SOURCE_CHANGED_AFTER_PROJECTION: %s",expected.File)
		}
	}
	return nil
}

func navigateGuarded(p projection, selector string, qualified []navigationSnapshot) (navigationResult,error) {
	paths:=make([]string,len(qualified))
	for i,s:=range qualified {paths[i]=s.File}
	selected,err:=navigate(p,selector,paths...)
	if err!=nil{return navigationResult{},err}
	if len(selected.SourceSnapshots)!=len(qualified) {
		return navigationResult{},fmt.Errorf("DFD_NAVIGATION_SOURCE_SNAPSHOT_COUNT_MISMATCH")
	}
	for i,s:=range qualified {
		if selected.SourceSnapshots[i]!=s {
			return navigationResult{},fmt.Errorf("DFD_REFERENCE_SOURCE_CHANGED_AFTER_PROJECTION: %s",s.File)
		}
	}
	if err:=verifySnapshots(qualified);err!=nil {return navigationResult{},err}
	return selected,nil
}
