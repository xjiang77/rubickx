package catalog_test

import (
	"encoding/json"
	"os"
	"testing"
)

var expectedOwners = []string{
	"Eng - End-to-End Network Path：从名称解析到应用响应",
	"Eng - IP Connectivity：IPv4、IPv6、Routing、NAT、ICMP 与 MTU",
	"Eng - Transport Semantics：TCP、UDP、QUIC、Flow Control 与 Backpressure",
	"Eng - DNS & Service Discovery：递归、缓存、分流与失败语义",
	"Eng - TLS & PKI：身份、信任链、mTLS 与证书生命周期",
	"Eng - Traffic Intermediation：Proxy、Gateway、Load Balancer、Tunnel 与 VPN",
	"Eng - SSO and Federation：OIDC、SAML、Kerberos 与协议选型",
	"Eng - Continuous Trust：PDP、PEP、Posture、Session、SSF 与 CAEP",
	"Eng - Enterprise Access Network：ZTNA、NAC、SSE、SASE、Segmentation 与 Egress",
	"Eng - Distributed Enforcement Lifecycle：Endpoint、DLP、Policy Revision 与 Effective State",
	"Eng - Network Observability：Packet、Flow、Log、Trace 与 Evidence",
	"Eng - Detection to Recovery：Telemetry、Signal、Decision、Action、Ack 与 Evidence",
}

type mapping struct {
	Owner    string `json:"owner"`
	Role     string `json:"role"`
	Coverage string `json:"coverage"`
}

type lab struct {
	ID              string    `json:"id"`
	Scope           string    `json:"scope"`
	Mappings        []mapping `json:"mappings"`
	BrowserRequired bool      `json:"browser_required"`
}

type catalog struct {
	SchemaVersion   int      `json:"schema_version"`
	KnowledgeOwners []string `json:"knowledge_owners"`
	Labs            []lab    `json:"labs"`
}

func TestCatalogV2OwnerAndScopeContract(t *testing.T) {
	raw, err := os.ReadFile("../../catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var got catalog
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 2 {
		t.Fatalf("schema_version=%d, want 2", got.SchemaVersion)
	}
	if len(got.KnowledgeOwners) != len(expectedOwners) {
		t.Fatalf("knowledge_owners=%d, want %d", len(got.KnowledgeOwners), len(expectedOwners))
	}
	owners := make(map[string]bool, len(expectedOwners))
	for index, owner := range expectedOwners {
		if got.KnowledgeOwners[index] != owner {
			t.Fatalf("knowledge_owners[%d]=%q, want %q", index, got.KnowledgeOwners[index], owner)
		}
		owners[owner] = true
	}
	if len(got.Labs) != 10 {
		t.Fatalf("labs=%d, want 10", len(got.Labs))
	}
	primary := make(map[string]bool, len(expectedOwners))
	for _, item := range got.Labs {
		wantScope := "core"
		if item.ID == "LAB-NETSEC-05" || item.ID == "LAB-NETSEC-06" {
			wantScope = "adjacent"
		}
		if item.Scope != wantScope {
			t.Errorf("%s scope=%q, want %q", item.ID, item.Scope, wantScope)
		}
		if wantScope == "adjacent" && len(item.Mappings) != 0 {
			t.Errorf("%s adjacent lab participates in canonical owner coverage", item.ID)
		}
		for _, link := range item.Mappings {
			if !owners[link.Owner] {
				t.Errorf("%s references non-canonical owner %q", item.ID, link.Owner)
			}
			if link.Role != "primary" && link.Role != "integration" {
				t.Errorf("%s owner %q role=%q", item.ID, link.Owner, link.Role)
			}
			if link.Role == "primary" {
				primary[link.Owner] = true
			}
			if link.Coverage != "executable" && link.Coverage != "model" {
				t.Errorf("%s owner %q coverage=%q", item.ID, link.Owner, link.Coverage)
			}
		}
	}
	for _, owner := range expectedOwners {
		if !primary[owner] {
			t.Errorf("owner %q has no primary lab", owner)
		}
	}
}

func TestCatalogCrossOwnerIntegrationMappings(t *testing.T) {
	raw, err := os.ReadFile("../../catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var got catalog
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	wanted := map[string]mapping{
		"LAB-NETSEC-08|Eng - SSO and Federation：OIDC、SAML、Kerberos 与协议选型": {
			Role: "integration", Coverage: "executable",
		},
		"LAB-NETSEC-10|Eng - Continuous Trust：PDP、PEP、Posture、Session、SSF 与 CAEP": {
			Role: "integration", Coverage: "executable",
		},
		"LAB-NETSEC-10|Eng - Enterprise Access Network：ZTNA、NAC、SSE、SASE、Segmentation 与 Egress": {
			Role: "integration", Coverage: "model",
		},
	}
	for _, item := range got.Labs {
		for _, link := range item.Mappings {
			key := item.ID + "|" + link.Owner
			if expected, ok := wanted[key]; ok && link.Role == expected.Role && link.Coverage == expected.Coverage {
				delete(wanted, key)
			}
		}
	}
	for key, expected := range wanted {
		t.Errorf("missing mapping %s role=%s coverage=%s", key, expected.Role, expected.Coverage)
	}
}

func TestCatalogBrowserRequiredExactSet(t *testing.T) {
	raw, err := os.ReadFile("../../catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var got catalog
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"LAB-NETSEC-05": true,
		"LAB-NETSEC-07": true,
		"LAB-NETSEC-08": true,
		"LAB-NETSEC-09": true,
	}
	for _, item := range got.Labs {
		if item.BrowserRequired != want[item.ID] {
			t.Errorf("%s browser_required=%t, want %t", item.ID, item.BrowserRequired, want[item.ID])
		}
	}
}
