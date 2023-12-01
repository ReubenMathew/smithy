package cloud

import (
	"encoding/json"
	"fmt"
)

type ComputeInstance struct {
	DnsName    string `json:"dns_name"`
	InstanceId string `json:"instance_id"`
}

type AgentCluster struct {
	SecurityGroupName string            `json:"security_group_name"`
	SecurityGroupId   string            `json:"security_group_id"`
	ComputeInstances  []ComputeInstance `json:"compute_instances"`
}

func (ac *AgentCluster) Bytes() []byte {
	bytes, err := json.Marshal(ac)
	if err != nil {
		panic(fmt.Sprintf("Failed to serialize agent-cluster metadata: %v", err))
	}
	return bytes
}
