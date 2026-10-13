package composition

// NewSelectedCoordinator bootstraps a host-owned route from an eligible
// Registry Selection. Unlike NewCoordinator, it never exposes an unselected
// router or skips contract eligibility during initial invocations. The caller
// has already prepared and attested initial and owns its Scope. It must not
// hand the Router or its guard to an untrusted plugin.
//
// As at cutover, the guard must be a nonblocking host-owned check which does
// not call back into the Registry (the registry mutex is held while calling it).
func NewSelectedCoordinator(registry *Registry, selected Selection, initial Binding, scope *Scope, guard func(Binding) error) (*Coordinator, error) {
	if registry == nil || guard == nil || scope == nil || !initial.valid() {
		return nil, ErrCoordinatorInvalid
	}
	selected.Dependencies = append([]Edge(nil), selected.Dependencies...)
	selected.DependencyEpochs = append([]ProviderEpoch(nil), selected.DependencyEpochs...)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if err := registry.validateSelectionLocked(selected, initial); err != nil {
		return nil, err
	}
	if err := guard(initial); err != nil {
		return nil, err
	}
	router, err := NewRouter(initial, scope)
	if err != nil {
		return nil, err
	}
	return &Coordinator{
		router: router, guard: guard, selectedRegistry: registry, selected: selected,
	}, nil
}
