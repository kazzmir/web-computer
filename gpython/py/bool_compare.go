package py

// bool is an int subtype for numeric comparisons, including reflected rich
// comparisons against Python-defined objects.
func (a Bool) M__lt__(v Object) (Object, error) { n, _ := a.M__index__(); return n.M__lt__(v) }
func (a Bool) M__le__(v Object) (Object, error) { n, _ := a.M__index__(); return n.M__le__(v) }
func (a Bool) M__gt__(v Object) (Object, error) { n, _ := a.M__index__(); return n.M__gt__(v) }
func (a Bool) M__ge__(v Object) (Object, error) { n, _ := a.M__index__(); return n.M__ge__(v) }
