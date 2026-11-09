package py

// Recursion accounting belongs to an interpreter context, not process-global
// state. Contexts already require serialized execution by their callers.
func (s *ModuleStore) RecursionLimit() int {
	if s.recursionLimit == 0 {
		return 1000
	}
	return s.recursionLimit
}
func (s *ModuleStore) RecursionDepth() int { return s.recursionDepth }
func (s *ModuleStore) SetRecursionLimit(limit int) error {
	if limit < 1 {
		return ExceptionNewf(ValueError, "recursion limit must be greater than zero")
	}
	if limit <= s.recursionDepth {
		return ExceptionNewf(RecursionError, "cannot set the recursion limit below the current recursion depth")
	}
	s.recursionLimit = limit
	return nil
}
func (s *ModuleStore) EnterFrame() error {
	if s.recursionDepth >= s.RecursionLimit() {
		return ExceptionNewf(RecursionError, "maximum recursion depth exceeded")
	}
	s.recursionDepth++
	return nil
}
func (s *ModuleStore) LeaveFrame() { s.recursionDepth-- }
