package types

type Diagnostic struct {
	Description string
	Condition   func(env *Environment) (skip bool, reason string)
	Run         func(env *Environment)
}
