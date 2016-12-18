// Package connection provides a service able to manage connections of the
// connection space.
package connection

import (
	"encoding/json"
	"fmt"
	"sync"

	storagecollection "github.com/the-anna-project/storage/collection"
	storageerror "github.com/the-anna-project/storage/error"
	"github.com/the-anna-project/worker"
)

// ServiceConfig represents the configuration used to create a new connection
// service.
type ServiceConfig struct {
	// Dependencies.
	StorageCollection *storagecollection.Collection
	WorkerService     worker.Service

	// Settings.
	Weight float64
}

// DefaultServiceConfig provides a default configuration to create a new
// connection service by best effort.
func DefaultServiceConfig() ServiceConfig {
	var err error

	var storageCollection *storagecollection.Collection
	{
		storageConfig := storagecollection.DefaultConfig()
		storageCollection, err = storagecollection.New(storageConfig)
		if err != nil {
			panic(err)
		}
	}

	var workerService worker.Service
	{
		workerConfig := worker.DefaultServiceConfig()
		workerService, err = worker.NewService(workerConfig)
		if err != nil {
			panic(err)
		}
	}

	config := ServiceConfig{
		// Dependencies.
		StorageCollection: storageCollection,
		WorkerService:     workerService,

		// Settings.
		Weight: 0,
	}

	return config
}

// NewService creates a new connection service.
func NewService(config ServiceConfig) (Service, error) {
	// Dependencies.
	if config.StorageCollection == nil {
		return nil, maskAnyf(invalidConfigError, "storage collection must not be empty")
	}
	if config.WorkerService == nil {
		return nil, maskAnyf(invalidConfigError, "worker service must not be empty")
	}

	// Settings.
	if config.Weight == 0 {
		return nil, maskAnyf(invalidConfigError, "weight must not be empty")
	}

	newService := &service{
		// Dependencies.
		storage: config.StorageCollection,
		worker:  config.WorkerService,

		// Internals.
		bootOnce:     sync.Once{},
		closer:       make(chan struct{}, 1),
		shutdownOnce: sync.Once{},

		// Settings.
		weight: config.Weight,
	}

	return newService, nil
}

type service struct {
	// Dependencies.
	storage *storagecollection.Collection
	worker  worker.Service

	// Internals.
	bootOnce     sync.Once
	closer       chan struct{}
	shutdownOnce sync.Once

	// Settings.
	weight float64
}

func (s *service) Boot() {
	s.bootOnce.Do(func() {
		// Service specific boot logic goes here.
	})
}

func (s *service) Create(namespaceA, namespaceB, peerAID, peerBID string) (Connection, error) {
	// In case the connection requested to be created already exists, we don't
	// need to do anything but return it.
	connection, err := s.Search(namespaceA, namespaceB, peerAID, peerBID)
	if IsNotFound(err) {
		// In case the peer does not exist, we can go ahead to create it.
	} else if err != nil {
		return nil, maskAny(err)
	}
	if connection != nil {
		// In case we found a connection, we return it. That way we are idempotent.
		return connection, nil
	}

	actions := []func(canceler <-chan struct{}) error{
		//
		func(canceler <-chan struct{}) error {
			listID := fmt.Sprintf("%s:%s:%s", namespaceA, namespaceB, peerAID)

			err := s.storage.Connection.PushToSet(listID, peerBID)
			if err != nil {
				return maskAny(err)
			}

			return nil
		},
		//
		func(canceler <-chan struct{}) error {
			connectionID := fmt.Sprintf("%s:%s:%s:%s", namespaceA, namespaceB, peerAID, peerBID)

			connectionConfig := DefaultConfig()
			connectionConfig.ID = connectionID
			connectionConfig.PeerAID = peerAID
			connectionConfig.PeerBID = peerBID
			connectionConfig.Weight = s.Weight()
			newConnection, err := New(connectionConfig)
			if err != nil {
				return maskAny(err)
			}
			connection = newConnection
			b, err := json.Marshal(newConnection)
			if err != nil {
				return maskAny(err)
			}
			err = s.storage.Connection.Set(connectionID, string(b))
			if err != nil {
				return maskAny(err)
			}

			return nil
		},
	}

	// Execute the list of actions asynchronously.
	executeConfig := s.worker.ExecuteConfig()
	executeConfig.Actions = actions
	executeConfig.Canceler = s.closer
	executeConfig.NumWorkers = len(actions)
	err = s.worker.Execute(executeConfig)
	if err != nil {
		return nil, maskAny(err)
	}

	return connection, nil
}

func (s *service) Delete(namespaceA, namespaceB, peerAID, peerBID string) error {
	// Check if the connection exists, to make sure we actually have to do
	// something.
	ok, err := s.Exists(namespaceA, namespaceB, peerAID, peerBID)
	if err != nil {
		return maskAny(err)
	}
	if !ok {
		return nil
	}

	actions := []func(canceler <-chan struct{}) error{
		func(canceler <-chan struct{}) error {
			listID := fmt.Sprintf("%s:%s:%s", namespaceA, namespaceB, peerAID)

			err := s.storage.Connection.RemoveFromSet(listID, peerBID)
			if err != nil {
				return maskAny(err)
			}

			return nil
		},
		func(canceler <-chan struct{}) error {
			connectionID := fmt.Sprintf("%s:%s:%s:%s", namespaceA, namespaceB, peerAID, peerBID)

			err := s.storage.Connection.Remove(connectionID)
			if err != nil {
				return maskAny(err)
			}

			return nil
		},
	}

	// Execute the list of actions asynchronously.
	executeConfig := s.worker.ExecuteConfig()
	executeConfig.Actions = actions
	executeConfig.Canceler = s.closer
	executeConfig.NumWorkers = len(actions)
	err = s.worker.Execute(executeConfig)
	if err != nil {
		return maskAny(err)
	}

	return nil
}

func (s *service) Exists(namespaceA, namespaceB, peerAID, peerBID string) (bool, error) {
	_, err := s.Search(namespaceA, namespaceB, peerAID, peerBID)
	if IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, maskAny(err)
	}

	return true, nil
}

func (s *service) Search(namespaceA, namespaceB, peerAID, peerBID string) (Connection, error) {
	connectionID := fmt.Sprintf("%s:%s:%s:%s", namespaceA, namespaceB, peerAID, peerBID)

	result, err := s.storage.Connection.Get(connectionID)
	if storageerror.IsNotFound(err) {
		return nil, maskAnyf(notFoundError, connectionID)
	} else if err != nil {
		return nil, maskAny(err)
	}

	newConnection, err := New(DefaultConfig())
	if err != nil {
		return nil, maskAny(err)
	}
	err = json.Unmarshal([]byte(result), newConnection)
	if err != nil {
		return nil, maskAny(err)
	}

	return newConnection, nil
}

func (s *service) SearchPeers(namespaceA, namespaceB, peerAID string) ([]string, error) {
	listID := fmt.Sprintf("%s:%s:%s", namespaceA, namespaceB, peerAID)

	result, err := s.storage.Connection.GetAllFromSet(listID)
	if err != nil {
		return nil, maskAny(err)
	}

	if len(result) == 0 {
		return nil, maskAnyf(notFoundError, listID)
	}

	return result, nil
}

func (s *service) Shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.closer)
	})
}

func (s *service) Weight() float64 {
	return s.weight
}
