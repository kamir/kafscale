// Copyright 2025 Alexander Alten (novatechflow), NovaTechflow (novatechflow.com).
// This project is supported and financed by Scalytics, Inc. (www.scalytics.io).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	"context"
	"testing"
	"time"

	"github.com/KafScale/platform/pkg/metadata"
	"github.com/KafScale/platform/pkg/protocol"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func TestRestartingDynamicMemberRetainsItsJoinIdentity(t *testing.T) {
	coord := NewGroupCoordinator(metadata.NewInMemoryStore(metadata.ClusterMetadata{}), protocol.MetadataBroker{}, &CoordinatorConfig{CleanupInterval: time.Hour})
	defer coord.Stop()
	coord.groups["test-group"] = &groupState{state: groupStateStable, generationID: 1, leaderID: "old-member", members: map[string]*memberState{"old-member": {sessionTimeout: time.Second, lastHeartbeat: time.Unix(0, 0), joinGeneration: 1}}, assignments: map[string][]assignmentTopic{}}
	req := kmsg.NewPtrJoinGroupRequest()
	req.Version = 4
	req.Group = "test-group"
	req.ProtocolType = "consumer"
	req.SessionTimeoutMillis = 45000
	req.RebalanceTimeoutMillis = 60000
	req.Protocols = []kmsg.JoinGroupRequestProtocol{{Name: "range", Metadata: coord.encodeSubscription([]string{"test-topic"})}}
	first, err := coord.JoinGroup(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ErrorCode != kerr.MemberIDRequired.Code || first.MemberID == "" {
		t.Fatalf("new member needs reusable identity: %+v", first)
	}
	req.MemberID = first.MemberID
	joined, err := coord.JoinGroup(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if joined.MemberID != first.MemberID {
		t.Fatal("member identity changed during rebalance")
	}
	coord.cleanupGroups()
	joined, err = coord.JoinGroup(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if joined.ErrorCode != 0 || joined.MemberID != first.MemberID {
		t.Fatalf("restart did not finish after stale member expiry: %+v", joined)
	}
}

func TestModernLeaveGroupRemovesListedMember(t *testing.T) {
	coord := NewGroupCoordinator(metadata.NewInMemoryStore(metadata.ClusterMetadata{}), protocol.MetadataBroker{}, &CoordinatorConfig{CleanupInterval: time.Hour})
	defer coord.Stop()
	coord.groups["test-group"] = &groupState{state: groupStateStable, generationID: 1, leaderID: "member", members: map[string]*memberState{"member": {sessionTimeout: time.Hour, lastHeartbeat: time.Now(), joinGeneration: 1}}, assignments: map[string][]assignmentTopic{}}
	req := kmsg.NewPtrLeaveGroupRequest()
	req.Version = 4
	req.Group = "test-group"
	req.Members = []kmsg.LeaveGroupRequestMember{{MemberID: "member"}}
	response := coord.LeaveGroup(context.Background(), req)
	if response.ErrorCode != 0 || len(response.Members) != 1 || response.Members[0].MemberID != "member" || response.Members[0].ErrorCode != 0 {
		t.Fatalf("modern leave response: %+v", response)
	}
	if coord.groups["test-group"] != nil {
		t.Fatal("departed member remains in group")
	}
}
