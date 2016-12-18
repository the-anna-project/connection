package connection

import (
	"encoding/json"
	"time"
)

type Connection interface {
	Created() time.Time
	ID() string
	json.Marshaler
	json.Unmarshaler
	PeerAID() string
	PeerBID() string
	Weight() float64
}

// TODO comment management of connections into one direction
type Service interface {
	Boot()
	Create(namespaceA, namespaceB, peerAID, peerBID string) (Connection, error)
	Delete(namespaceA, namespaceB, peerAID, peerBID string) error
	Exists(namespaceA, namespaceB, peerAID, peerBID string) (bool, error)
	Search(namespaceA, namespaceB, peerAID, peerBID string) (Connection, error)
	SearchPeers(namespaceA, namespaceB, peerAID string) ([]string, error)
	Shutdown()
	Weight() float64
}
