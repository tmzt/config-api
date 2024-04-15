package util

type ScopeKind string

const (
	ScopeKindInvalid ScopeKind = "invalid"

	ScopeKindGlobal  ScopeKind = "global"
	ScopeKindAccount ScopeKind = "account"
	ScopeKindUser    ScopeKind = "user"
)

func ScopeKindPtr(s string) *ScopeKind {
	v := ScopeKind(s)
	return &v
}

func ScopeKindAsPtr(s ScopeKind) *ScopeKind {
	return &s
}
