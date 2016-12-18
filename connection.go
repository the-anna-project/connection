package connection

import (
	"time"
)

// Config represents the configuration used to create a new connection.
type Config struct {
	// Settings.
	Created time.Time
	ID      string
	PeerAID string
	PeerBID string
	Weight  float64
}

// DefaultConfig provides a default configuration to create a new connection by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Settings.
		Created: time.Now(),
		ID:      "",
		PeerAID: "",
		PeerBID: "",
		Weight:  0,
	}
}

// New creates a new configured connection.
func New(config Config) (Connection, error) {
	// Dependencies.
	if config.Created.IsZero() {
		return nil, maskAnyf(invalidConfigError, "created must not be empty")
	}
	if config.ID == "" {
		return nil, maskAnyf(invalidConfigError, "ID must not be empty")
	}
	if config.PeerAID == "" {
		return nil, maskAnyf(invalidConfigError, "peerA ID must not be empty")
	}
	if config.PeerBID == "" {
		return nil, maskAnyf(invalidConfigError, "peerB ID must not be empty")
	}

	newConnection := &connection{
		// Settings.
		created: config.Created,
		id:      config.ID,
		peerAID: config.PeerAID,
		peerBID: config.PeerBID,
		weight:  config.Weight,
	}

	return newConnection, nil
}

type connection struct {
	// Settings.
	created time.Time
	id      string
	peerAID string
	peerBID string
	weight  float64
}

func (c *connection) Created() time.Time {
	return c.created
}

func (c *connection) ID() string {
	return c.id
}

func (c *connection) PeerAID() string {
	return c.peerAID
}

func (c *connection) PeerBID() string {
	return c.peerBID
}

func (c *connection) Weight() float64 {
	return c.weight
}
