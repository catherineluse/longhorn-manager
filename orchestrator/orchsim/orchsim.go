package orchsim

import (
	"fmt"
	"sync"

	"github.com/yasker/lm-rewrite/orchestrator"
	"github.com/yasker/lm-rewrite/types"
	"github.com/yasker/lm-rewrite/util"
)

type OrchSim struct {
	hostID  string
	records map[string]*InstanceRecord
	mutex   *sync.RWMutex
}

type StateType string

const (
	StateRunning = StateType("running")
	StateStopped = StateType("stopped")
)

type InstanceRecord struct {
	ID    string
	Name  string
	State StateType
	IP    string
}

func NewOrchestratorSimulator(hostID string) (orchestrator.Orchestrator, error) {
	return &OrchSim{
		hostID:  hostID,
		records: map[string]*InstanceRecord{},
		mutex:   &sync.RWMutex{},
	}, nil
}

func (s *OrchSim) CreateController(request *orchestrator.Request) (*types.ControllerInfo, error) {
	if request.HostID != s.GetCurrentHostID() {
		return nil, fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}
	if request.InstanceName == "" {
		return nil, fmt.Errorf("missing required field %+v", request)
	}

	instance := &InstanceRecord{
		ID:    util.UUID(),
		Name:  request.InstanceName,
		State: StateRunning,
		IP:    "ip-" + request.InstanceName + "-" + util.UUID()[:8],
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.createRecord(instance); err != nil {
		return nil, err
	}
	return &types.ControllerInfo{
		InstanceInfo: types.InstanceInfo{
			ID:      instance.ID,
			Name:    instance.Name,
			HostID:  s.GetCurrentHostID(),
			Address: instance.IP,
			Running: instance.State == StateRunning,
		},
	}, nil
}

func (s *OrchSim) CreateReplica(request *orchestrator.Request) (*types.ReplicaInfo, error) {
	if request.HostID != s.GetCurrentHostID() {
		return nil, fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}
	if request.InstanceName == "" {
		return nil, fmt.Errorf("missing required field %+v", request)
	}

	instance := &InstanceRecord{
		ID:    util.UUID(),
		Name:  request.InstanceName,
		State: StateStopped,
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.createRecord(instance); err != nil {
		return nil, err
	}
	return &types.ReplicaInfo{
		InstanceInfo: types.InstanceInfo{
			ID:      instance.ID,
			Name:    instance.Name,
			HostID:  s.GetCurrentHostID(),
			Address: instance.IP,
			Running: instance.State == StateRunning,
		},

		Mode:         "",
		BadTimestamp: "",
	}, nil
}

func (s *OrchSim) StartInstance(request *orchestrator.Request) (*types.InstanceInfo, error) {
	if request.HostID != s.GetCurrentHostID() {
		return nil, fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}

	if request.InstanceName == "" {
		return nil, fmt.Errorf("missing required field %+v", request)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	instance, err := s.getRecord(request.InstanceName)
	if err != nil {
		return nil, err
	}
	if instance.State != StateRunning {
		instance.State = StateRunning
		instance.IP = "ip-" + instance.Name + "-" + util.UUID()[:8]
		if err := s.updateRecord(instance); err != nil {
			return nil, err
		}
	}
	return &types.InstanceInfo{
		ID:      instance.ID,
		Name:    instance.Name,
		HostID:  s.GetCurrentHostID(),
		Address: instance.IP,
		Running: instance.State == StateRunning,
	}, nil
}

func (s *OrchSim) StopInstance(request *orchestrator.Request) (*types.InstanceInfo, error) {
	if request.HostID != s.GetCurrentHostID() {
		return nil, fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}
	if request.InstanceName == "" {
		return nil, fmt.Errorf("missing required field %+v", request)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	instance, err := s.getRecord(request.InstanceName)
	if err != nil {
		return nil, err
	}
	if instance.State != StateStopped {
		instance.State = StateStopped
		instance.IP = ""
		if err := s.updateRecord(instance); err != nil {
			return nil, err
		}
	}
	return &types.InstanceInfo{
		ID:      instance.ID,
		Name:    instance.Name,
		HostID:  s.GetCurrentHostID(),
		Address: instance.IP,
		Running: instance.State == StateRunning,
	}, nil
}

func (s *OrchSim) RemoveInstance(request *orchestrator.Request) error {
	if request.HostID != s.GetCurrentHostID() {
		return fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.removeRecord(request.InstanceName)
}

func (s *OrchSim) InspectInstance(request *orchestrator.Request) (*types.InstanceInfo, error) {
	if request.HostID != s.GetCurrentHostID() {
		return nil, fmt.Errorf("incorrect host, requested %v, current %v", request.HostID,
			s.GetCurrentHostID())
	}
	if request.InstanceName == "" {
		return nil, fmt.Errorf("missing required field %+v", request)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	instance, err := s.getRecord(request.InstanceName)
	if err != nil {
		return nil, err
	}
	return &types.InstanceInfo{
		ID:      instance.ID,
		Name:    instance.Name,
		HostID:  s.GetCurrentHostID(),
		Address: instance.IP,
		Running: instance.State == StateRunning,
	}, nil
}

func (s *OrchSim) GetCurrentHostID() string {
	return s.hostID
}

// Must be locked
func (s *OrchSim) createRecord(instance *InstanceRecord) error {
	if s.records[instance.Name] != nil {
		return fmt.Errorf("duplicate instance with name %v", instance.Name)
	}
	s.records[instance.Name] = instance
	return nil
}

// Must be locked
func (s *OrchSim) updateRecord(instance *InstanceRecord) error {
	if s.records[instance.Name] == nil {
		return fmt.Errorf("unable to find instance with name %v", instance.Name)
	}
	s.records[instance.Name] = instance
	return nil
}

// Must be locked
func (s *OrchSim) getRecord(instanceName string) (*InstanceRecord, error) {
	if s.records[instanceName] == nil {
		return nil, fmt.Errorf("unable to find instance %v", instanceName)
	}
	return s.records[instanceName], nil
}

// Must be locked
func (s *OrchSim) removeRecord(instanceName string) error {
	if s.records[instanceName] == nil {
		return fmt.Errorf("unable to find instance %v", instanceName)
	}
	delete(s.records, instanceName)
	return nil
}