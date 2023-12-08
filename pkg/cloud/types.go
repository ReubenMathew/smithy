package cloud

import (
	"encoding/json"
	"fmt"
)

type ComputeInstance struct {
	DnsName    string `json:"dns_name"`
	InstanceId string `json:"instance_id"`
	PrivateIp  string `json:"private_ip"`
	PublicIp   string `json:"public_ip"`
}

type AgentCluster struct {
	SecurityGroupName string            `json:"security_group_name"`
	SecurityGroupId   string            `json:"security_group_id"`
	ComputeInstances  []ComputeInstance `json:"compute_instances"`
}

func LoadAgentCluster(bytes []byte) (*AgentCluster, error) {
	var ac AgentCluster
	if err := json.Unmarshal(bytes, &ac); err != nil {
		return nil, err
	}
	return &ac, nil
}

func (ac *AgentCluster) Bytes() []byte {
	bytes, err := json.Marshal(ac)
	if err != nil {
		panic(fmt.Sprintf("Failed to serialize agent-cluster metadata: %v", err))
	}
	return bytes
}
