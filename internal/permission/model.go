package permission

import (
	"fmt"
	"sort"

	"tei/internal/core"
)

type Kind string

const (
	KindFilesystem Kind = "filesystem"
	KindNetwork    Kind = "network"
	KindSecret     Kind = "secret"
)

type Access string

const (
	AccessRead      Access = "read"
	AccessWrite     Access = "write"
	AccessReadWrite Access = "read-write"
	AccessConnect   Access = "connect"
)

type Spec struct {
	Kind   Kind   `json:"kind"`
	Access Access `json:"access"`
	Scope  string `json:"scope"`
}

func (s Spec) Validate() error {
	if s.Scope == "" {
		return fmt.Errorf("permission scope is required")
	}
	switch s.Kind {
	case KindFilesystem:
		if s.Access != AccessRead && s.Access != AccessWrite && s.Access != AccessReadWrite {
			return fmt.Errorf("filesystem permission access must be read, write, or read-write")
		}
	case KindNetwork:
		if s.Access != AccessConnect {
			return fmt.Errorf("network permission access must be connect")
		}
	case KindSecret:
		if s.Access != AccessRead {
			return fmt.Errorf("secret permission access must be read")
		}
	default:
		return fmt.Errorf("unknown permission kind %q", s.Kind)
	}
	return nil
}

func (s Spec) Key() string { return string(s.Kind) + "|" + string(s.Access) + "|" + s.Scope }

type Grant struct {
	CapabilityKind string           `json:"capabilityKind"`
	CapabilityID   core.CanonicalID `json:"capabilityId"`
	Permission     Spec             `json:"permission"`
}

func Normalize(in []Spec) []Spec {
	out := append([]Spec(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}
