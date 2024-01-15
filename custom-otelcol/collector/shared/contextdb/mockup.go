package contextdb

var appdData = []map[string]interface{}{
	{"application": "Mockup-App", "tier": "Mock-Tier-1", "node": "node1", "ipv4": []string{"10.133.10.150", "10.134.10.150"}},
	{"application": "Mockup-App", "tier": "Mock-Tier-1", "node": "node2", "ipv4": []string{"10.133.10.151", "10.134.10.151"}},
	{"application": "Mockup-App", "tier": "Mock-Tier-2", "node": "node3", "ipv4": []string{"10.133.10.152", "10.134.10.152"}},
	{"application": "Mockup-App", "tier": "Mock-Tier-2", "node": "node4", "ipv4": []string{"10.133.10.153", "10.134.10.153"}},
	{"application": "Mockup-App", "tier": "Mock-Tier-3", "node": "node5", "ipv4": []string{"10.133.10.154", "10.134.10.154"}},

	{"application": "Mockup-Cont", "tier": "Cont-Tier-1", "node": "cont1", "ipv4": []string{"10.10.10.150"}},
	{"application": "Mockup-Cont", "tier": "Cont-Tier-1", "node": "cont2", "ipv4": []string{"10.10.10.151", "10.133.10.154"}},
	{"application": "Mockup-Cont", "tier": "Cont-Tier-2", "node": "cont3", "ipv4": []string{"10.10.10.152"}},
	{"application": "Mockup-Cont", "tier": "Cont-Tier-2", "node": "cont4", "ipv4": []string{"10.10.10.153"}},
	{"application": "Mockup-Cont", "tier": "Cont-Tier-3", "node": "cont5", "ipv4": []string{"10.10.10.154", "10.133.10.154"}},
}

var k8sData = []map[string]interface{}{
	{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand12345678", "ipv4": "10.10.10.150"},
	{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand87654321", "ipv4": "10.10.10.151"},
	{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand11111111", "ipv4": "10.10.10.152"},
	{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand22222222", "ipv4": "10.10.10.153"},
	{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand33333333", "ipv4": "10.10.10.154"},
}
