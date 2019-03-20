package metrics

import "sync"

// GaugeFloat64 holds a float64 Value that can be set arbitrarily.
type GaugeFloat64 interface {
	Snapshot() GaugeFloat64
	Update(float64)
	UpdateWithTag(float64, Tag)
	Value() float64
	Tag() *Tag
	IsSet() bool
	Clear()
}

// GetOrRegisterGaugeFloat64 returns an existing GaugeFloat64 or constructs and registers a
// new StandardGaugeFloat64.
func GetOrRegisterGaugeFloat64(name string, registry Registry) GaugeFloat64 {
	if nil == registry {
		registry = DefaultRegistry
	}
	return registry.GetOrRegister(name, NewGaugeFloat64()).(GaugeFloat64)
}

// NewGaugeFloat64 constructs a new StandardGaugeFloat64.
func NewGaugeFloat64() GaugeFloat64 {
	if UseNilMetrics {
		return NilGaugeFloat64{}
	}
	return &StandardGaugeFloat64{
		value: 0.0,
		tag: nil,
		isSet: false,
	}
}

// NewRegisteredGaugeFloat64 constructs and registers a new StandardGaugeFloat64.
func NewRegisteredGaugeFloat64(name string, registry Registry) GaugeFloat64 {
	gauge := NewGaugeFloat64()
	if nil == registry {
		registry = DefaultRegistry
	}
	LogErrorIfAny(registry.Register(name, gauge))
	return gauge
}

// NewFunctionalGaugeFloat64 constructs a new FunctionalGauge.
func NewFunctionalGaugeFloat64(value func() float64, tag func() *Tag, isSet func() bool) GaugeFloat64 {
	if UseNilMetrics {
		return NilGaugeFloat64{}
	}
	return &FunctionalGaugeFloat64{
		value: value,
		tag:   tag,
		isSet: isSet,
	}
}

// NewRegisteredFunctionalGaugeFloat64 constructs and registers a new StandardGauge.
func NewRegisteredFunctionalGaugeFloat64(name string, registry Registry, value func() float64, tag func() *Tag, isSet func() bool) GaugeFloat64 {
	gauge := NewFunctionalGaugeFloat64(value, tag, isSet)
	if nil == registry {
		registry = DefaultRegistry
	}
	LogErrorIfAny(registry.Register(name, gauge))
	return gauge
}

// GaugeFloat64Snapshot is a read-only copy of another GaugeFloat64.
type GaugeFloat64Snapshot struct {
	value float64
	tag *Tag
	isSet bool
}

// Snapshot returns the snapshot.
func (gauge GaugeFloat64Snapshot) Snapshot() GaugeFloat64 { return gauge }

// Update panics.
func (GaugeFloat64Snapshot) Update(float64) {
	panic("Update called on a GaugeFloat64Snapshot")
}

// UpdateWithTag panics.
func (GaugeFloat64Snapshot) UpdateWithTag(float64, Tag) {
	panic("UpdateWithTag called on a GaugeFloat64Snapshot")
}


// Value returns the Value at the time the snapshot was taken.
func (gauge GaugeFloat64Snapshot) Value() float64 {
	return gauge.value
}

// Value returns the tag at the time the snapshot was taken.
func (gauge GaugeFloat64Snapshot) Tag() *Tag {
	return gauge.tag
}

// IsSet returns the isSet at the time the snapshot was taken.
func (gauge GaugeFloat64Snapshot) IsSet() bool {
	return gauge.isSet
}

// Clear is not supposed to call for GaugeFloat64Snapshot.
func (gauge GaugeFloat64Snapshot) Clear() {
	panic("Clear called on a GaugeFloat64Snapshot")
}

// NilGaugeFloat64 is a no-op Gauge.
type NilGaugeFloat64 struct{}

// Snapshot is a no-op.
func (NilGaugeFloat64) Snapshot() GaugeFloat64 { return NilGaugeFloat64{} }

// Update is a no-op.
func (NilGaugeFloat64) Update(value float64) {}

// UpdateWithTag is a no-op.
func (NilGaugeFloat64) UpdateWithTag(value float64, tag Tag) {}

// Value is a no-op.
func (NilGaugeFloat64) Value() float64 { return 0.0 }

// Tag is a no-op.
func (NilGaugeFloat64) Tag() *Tag { return nil }

// IsSet is a no-op.
func (NilGaugeFloat64) IsSet() bool { return false }

// Clear is a no-op.
func (NilGaugeFloat64) Clear() {}

// StandardGaugeFloat64 is the standard implementation of a GaugeFloat64 and uses
// sync.Mutex to manage the struct values.
type StandardGaugeFloat64 struct {
	mutex sync.Mutex
	value float64
	tag *Tag
	isSet bool
}

// Snapshot returns a read-only copy of the gauge.
func (gauge *StandardGaugeFloat64) Snapshot() GaugeFloat64 {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	tag:= gauge.tag
	if tag != nil {
		tag = &Tag{Name: gauge.tag.Name, Value: gauge.tag.Value}
	}
	return GaugeFloat64Snapshot{gauge.value, tag, gauge.isSet}
}

// Update updates the gauge's Value.
func (gauge *StandardGaugeFloat64) Update(value float64) {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = value
	gauge.tag = nil
	gauge.isSet = true
}

// Update updates the gauge's Value and corresponding tag
func (gauge *StandardGaugeFloat64) UpdateWithTag(value float64, tag Tag) {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = value
	gauge.tag = &Tag{Name: tag.Name, Value: tag.Value}
	gauge.isSet = true
}

// Value returns the gauge's current Value.
func (gauge *StandardGaugeFloat64) Value() float64 {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.value
}

// Value returns the gauge's current tag.
func (gauge *StandardGaugeFloat64) Tag() *Tag {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.tag
}

// IsSet returns the gauge's isSet Value.
func (gauge *StandardGaugeFloat64) IsSet() bool {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.isSet
}

// Clear resets the gauge's Value to the defaults
func (gauge *StandardGaugeFloat64) Clear() {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = 0.0
	gauge.tag = nil
	gauge.isSet = false
}

// FunctionalGaugeFloat64 returns Value from given function
type FunctionalGaugeFloat64 struct {
	value func() float64
	tag func() *Tag
	isSet func() bool
}

// Value returns the gauge's current Value.
func (gauge FunctionalGaugeFloat64) Value() float64 {
	return gauge.value()
}

// Value returns the gauge's current tag.
func (gauge FunctionalGaugeFloat64) Tag() *Tag {
	return gauge.tag()
}

// IsSet returns the gauge's isSet Value.
func (gauge FunctionalGaugeFloat64) IsSet() bool {
	return gauge.isSet()
}

// Snapshot returns the snapshot.
func (gauge FunctionalGaugeFloat64) Snapshot() GaugeFloat64 {
	return GaugeFloat64Snapshot{
		gauge.Value(),
		gauge.Tag(),
		gauge.IsSet(),
	}
}

// Update panics.
func (FunctionalGaugeFloat64) Update(float64) {
	panic("Update called on a FunctionalGaugeFloat64")
}

// UpdateWithTag panics.
func (FunctionalGaugeFloat64) UpdateWithTag(float64, Tag) {
	panic("UpdateWithTag called on a FunctionalGaugeFloat64")
}

// Clear is not supposed to call on FunctionalGaugeFloat64.
func (FunctionalGaugeFloat64) Clear() {
	panic("Clear called on a FunctionalGaugeFloat64")
}
