package messagelocalization

import (
 "path/filepath"
 "runtime"
 "strings"
 "testing"
 "tei/internal/plugin"
 "tei/internal/workspace"
)

func TestWorkspaceAdapterRealSourceAndMissingMessage(t *testing.T) {
 _, file, _, ok := runtime.Caller(0)
 if !ok {t.Fatal("runtime.Caller failed")}
 root:=filepath.Join(filepath.Dir(file),"..","workspace")
 loaded,err:=workspace.Load(root)
 if err!=nil {t.Fatal(err)}
 lock,err:=plugin.BuildLock(plugin.Catalog{Plugins:loaded.World.Plugins,Bundles:loaded.World.Bundles},loaded.World.Environments)
 if err!=nil {t.Fatal(err)}
 req:=WorkspaceRequest{Root:root,Environment:"development",Lock:lock,Review:func(ReviewInput)error{return nil}}
 if _,err:=ResolveWorkspace(req);err==nil || !strings.Contains(err.Error(),"std-message-unselected"){t.Fatalf("unexpected workspace resolution: %v",err)}
 req.Environment="absent"
 if _,err:=ResolveWorkspace(req);err==nil || !strings.Contains(err.Error(),"missing-environment"){t.Fatalf("missing environment accepted: %v",err)}
 req.Environment="development"
 req.Lock=plugin.Lock{}
 if _,err:=ResolveWorkspace(req);err==nil || !strings.Contains(err.Error(),"plugin-lock"){t.Fatalf("missing lock accepted: %v",err)}
}

func TestWorkspaceAdapterRequiresHostReview(t *testing.T) {
 req:=WorkspaceRequest{Root:"absent",Environment:"development"}
 if _,err:=ResolveWorkspace(req);err==nil || !strings.Contains(err.Error(),"host-review-required"){t.Fatalf("review not required: %v",err)}
 req.Review=func(ReviewInput)error{return nil}
 if _,err:=ResolveWorkspace(req);err==nil || !strings.Contains(err.Error(),"load:"){t.Fatalf("missing workspace accepted: %v",err)}
}
