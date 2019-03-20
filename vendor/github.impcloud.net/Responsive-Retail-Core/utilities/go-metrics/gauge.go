package metrics

import (
	"sync"
)

// Gauge holds an int64 Value that can be set arbitrarily.
type Gauge interface {
	Snapshot() Gauge
	Update(int64)
	UpdateWithTag(int64, Tag)
	Value() int64
	Tag() *Tag
	IsSet() bool
	Clear()
}

// GetOrRegisterGauge returns an existing Gauge or constructs and registers a
// new StandardGauge.
func GetOrRegisterGauge(name string, registry Registry) Gauge {
	if nil == registry {
		registry = DefaultRegistry
	}
	return registry.GetOrRegister(name, NewGauge).(Gauge)
}

// NewGauge constructs a new StandardGauge.
func NewGauge() Gauge {
	if UseNilMetrics {
		return NilGauge{}
	}
	return &StandardGauge{
		value: 0,
		tag: nil,
		isSet: false,
	}
}

// NewRegisteredGauge constructs and registers a new StandardGauge.
func NewRegisteredGauge(name string, registry Registry) Gauge {
	gauge := NewGauge()
	if nil == registry {
		registry = DefaultRegistry
	}
	LogErrorIfAny(registry.Register(name, gauge))
	return gauge
}

// NewFunctionalGauge constructs a new FunctionalGauge.
func NewFunctionalGauge(value func() int64, tag func() *Tag,  isSet func() bool) Gauge {
	if UseNilMetrics {
		return NilGauge{}
	}
	return &FunctionalGauge{
		value: value,
		tag:   tag,
		isSet: isSet,
	}
}

// NewRegisteredFunctionalGauge constructs and registers a new StandardGauge.
func NewRegisteredFunctionalGauge(name string, registry Registry, value func() int64, tag func() *Tag, isSet func() bool) Gauge {
	gauge := NewFunctionalGauge(value, tag, isSet)
	if nil == registry {
		registry = DefaultRegistry
	}
	LogErrorIfAny(registry.Register(name, gauge))
	return gauge
}

// GaugeSnapshot is a read-only copy of another Gauge.
type GaugeSnapshot struct {
	value int64
	tag *Tag
	isSet bool
}

// Snapshot returns the snapshot.
func (gauge GaugeSnapshot) Snapshot() Gauge { return gauge }

// Update panics.
func (GaugeSnapshot) Update(int64) {
	panic("Update called on a GaugeSnapshot")
}

// Update panics.
func (GaugeSnapshot) UpdateWithTag(int64, Tag) {
	panic("UpdateWithTag called on a GaugeSnapshot")
}

// Value returns the Value at the time the snapshot was taken.
func (gauge GaugeSnapshot) Value() int64 {
	return gauge.value
}

// Value returns the tag at the time the snapshot was taken.
func (gauge GaugeSnapshot) Tag() *Tag {
	return gauge.tag
}

// IsSet returns whether gauge snapshot is set
func (gauge GaugeSnapshot) IsSet() bool {
	return gauge.isSet
}

// Clear is not supposed to call for GaugeSnapshot
func (gauge GaugeSnapshot) Clear() {
	panic("Clear called on a GaugeSnapshot")
}

// NilGauge is a no-op Gauge.
type NilGauge struct{}

// Snapshot is a no-op.
func (NilGauge) Snapshot() Gauge { return NilGauge{} }

// Update is a no-op.
func (NilGauge) Update(value int64) {}

// UpdateWithTag is a no-op.
func (NilGauge) UpdateWithTag(int64, Tag) {}

// Value is a no-op.
func (NilGauge) Value() int64 { return 0 }

// Tag is a no-op.
func (NilGauge) Tag() *Tag { return  nil}

// IsSet is a no-op.
func (NilGauge) IsSet() bool { return false }

// Clear is a no-op.
func (NilGauge) Clear() {}


// StandardGauge is the standard implementation of a Gauge and uses the
// sync.Mutex to manage the struct values.
type StandardGauge struct {
	mutex sync.Mutex
	value int64
	tag *Tag
	isSet bool
}

// Snapshot returns a read-only copy of the gauge.
func (gauge *StandardGauge) Snapshot() Gauge {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	tag:= gauge.tag
	if tag != nil {
		tag = &Tag{Name:gauge.tag.Name, Value:gauge.tag.Value}
	}
	return GaugeSnapshot{gauge.value, tag, gauge.isSet, }
}

// Update updates the gauge's Value.
func (gauge *StandardGauge) Update(value int64) {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = value
	gauge.tag = nil
	gauge.isSet = true
}

// Update updates the gauge's Value with a corresponding tag
func (gauge *StandardGauge) UpdateWithTag(value int64, tag Tag) {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = value
	gauge.tag = &Tag{Name:tag.Name, Value:tag.Value}
	gauge.isSet = true
}

// Value returns the gauge's current Value.
func (gauge *StandardGauge) Value() int64 {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.value
}

// Value returns the gauge's current Tag.
func (gauge *StandardGauge) Tag() *Tag {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.tag
}

// IsSet returns whether standard gauge is set
func (gauge *StandardGauge) IsSet() bool {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	return gauge.isSet
}

// Clear reset the standard gauge to its default state
func (gauge *StandardGauge) Clear() {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()
	gauge.value = 0
	gauge.tag = nil
	gauge.isSet = false
}

// FunctionalGauge returns Value from given function
type FunctionalGauge struct {
	value func() int64
	tag func() *Tag
	isSet func() bool
}

// Value returns the gauge's current Value.
func (g FunctionalGauge) Value() int64 {
	return g.value()
}

// Value returns the gauge's current tag.
func (g FunctionalGauge) Tag() *Tag {
	return g.tag()
}

// IsSet returns whether functional gauge is set
func (g FunctionalGauge) IsSet() bool {
	return g.isSet()
}

// Snapshot returns the snapshot.
func (g FunctionalGauge) Snapshot() Gauge {
	return GaugeSnapshot{
		g.Value(),
		g.tag(),
		g.IsSet(),
	}
}

// Update panics.
func (FunctionalGauge) Update(int64) {
	panic("Update called on a FunctionalGauge")
}

// UpdateWithTag panics.
func (FunctionalGauge) UpdateWithTag(int64, Tag) {
	panic("UpdateWithTag called on a FunctionalGauge")
}

// Clear is not supposed to call for FunctionalGauge
func (FunctionalGauge) Clear() {
	panic("Clear called on a FunctionalGauge")
}
