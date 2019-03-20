package metrics

import (
	"sync"
	"time"
)

// GaugeCollection holds a collection of int64 values and timestamps that can be accumulated.
type GaugeCollection interface {
	Snapshot() GaugeCollection
	Add(int64)
	AddWithTag(int64, Tag)
	Readings() []GaugeReading
	IsSet() bool
	Clear()
}

type GaugeReading struct {
	Reading     int64
	Tag         *Tag
	Time        time.Time
}

// GetOrRegisterGaugeCollection returns an existing Gauge or constructs and registers a
// new StandardGaugeCollection.
func GetOrRegisterGaugeCollection(name string, registry Registry) GaugeCollection {
	if nil == registry {
		registry = DefaultRegistry
	}
	return registry.GetOrRegister(name, NewGaugeCollection).(GaugeCollection)
}

// NewGaugeCollection constructs a new StandardGaugeCollection.
func NewGaugeCollection() GaugeCollection {
	if UseNilMetrics {
		return NilGaugeCollection{}
	}
	return &StandardGaugeCollection{}
}

// NewRegisteredGaugeCollection constructs and registers a new StandardGaugeCollection.
func NewRegisteredGaugeCollection(name string, registry Registry) GaugeCollection {
	gaugeCollection := NewGaugeCollection()
	if nil == registry {
		registry = DefaultRegistry
	}
	LogErrorIfAny(registry.Register(name, gaugeCollection))
	return gaugeCollection
}

// GaugeCollectionSnapshot is a read-only copy of another GaugeCollection.
type GaugeCollectionSnapshot struct {
	readings []GaugeReading
}

// Snapshot returns the snapshot.
func (gaugeCollection GaugeCollectionSnapshot) Snapshot() GaugeCollection { return gaugeCollection }

// Add panics. Suppose to be read-only
func (GaugeCollectionSnapshot) Add(int64) {
	panic("Add called on a GaugeCollectionSnapshot")
}

// AddWithTag panics. Suppose to be read-only
func (GaugeCollectionSnapshot) AddWithTag(int64, Tag) {
	panic("AddWithTag called on a GaugeCollectionSnapshot")
}

// Readings returns the collect of readings at the time the snapshot was taken.
func (gaugeCollection GaugeCollectionSnapshot) Readings() []GaugeReading {
	return gaugeCollection.readings
}

// IsSet returns true if the collection has any values
func (gaugeCollection GaugeCollectionSnapshot) IsSet() bool {
	return len(gaugeCollection.readings) > 0
}

func (gaugeCollection GaugeCollectionSnapshot) Clear() {
	panic("Clear called on a GaugeSnapshot")
}

// NilGauge is a no-op Gauge.
type NilGaugeCollection struct{}

// Snapshot is a no-op.
func (NilGaugeCollection) Snapshot() GaugeCollection { return NilGaugeCollection{} }

// Add is a no-op.
func (NilGaugeCollection) Add(value int64) {}

// AddWithTag is a no-op.
func (NilGaugeCollection) AddWithTag(value int64, tag Tag) {}

// Readings is a no-op.
func (NilGaugeCollection) Readings() []GaugeReading { return nil }

// IsSet is a no-op.
func (NilGaugeCollection) IsSet() bool { return false}

// Clear is a no-op.
func (NilGaugeCollection) Clear() { }

// StandardGaugeCollection is the standard implementation of a GaugeCollection and uses the
// sync.Mutex to manage the struct values.
type StandardGaugeCollection struct {
	mutex sync.Mutex
	readings []GaugeReading
}

// Snapshot returns a read-only copy of the GaugeCollection.
func (gaugeCollection *StandardGaugeCollection) Snapshot() GaugeCollection {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	readings :=	make([]GaugeReading, len(gaugeCollection.readings))
	copy(readings, gaugeCollection.readings)
	return GaugeCollectionSnapshot {readings}
}

// Add Adds a reading to the collection.
func (gaugeCollection *StandardGaugeCollection) Add(reading int64) {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	gaugeReading := GaugeReading{ Reading: reading, Time: time.Now()}
	gaugeCollection.readings = append(gaugeCollection.readings, gaugeReading)
}

// AddWithTag Adds a reading to the collection with a corresponding tag
func (gaugeCollection *StandardGaugeCollection) AddWithTag(reading int64, tag Tag) {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	gaugeReading := GaugeReading {
		Reading: reading,
		Tag: &Tag{
			Name:  tag.Name,
			Value: tag.Value,
		},
		Time: time.Now(),
	}
	gaugeCollection.readings = append(gaugeCollection.readings, gaugeReading)
}


// Readings returns a copy of the collection of readings.
func (gaugeCollection *StandardGaugeCollection) Readings() []GaugeReading {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	readings :=	make([]GaugeReading, len(gaugeCollection.readings))
	copy(readings, gaugeCollection.readings)
	return readings
}

// IsSet returns true if the collection has any values
func (gaugeCollection *StandardGaugeCollection) IsSet() bool {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	return len(gaugeCollection.readings) > 0
}

func (gaugeCollection *StandardGaugeCollection) Clear() {
	gaugeCollection.mutex.Lock()
	defer gaugeCollection.mutex.Unlock()
	gaugeCollection.readings = []GaugeReading{}
}

