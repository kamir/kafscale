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

package main

import (
	"reflect"
	"testing"

	"github.com/twmb/franz-go/pkg/kmsg"
)

// These two tests come from the independent fix in PR #21 by @jay23606 and are kept
// alongside the ones in kmsg_defaults_test.go because they cover a different thing:
// they call the real entry points addErrorForAllPartitions and
// addFetchErrorForAllPartitions, not the newProduceErrorPartition and
// newFetchErrorPartition helpers. A helper-only test stays green if someone later
// builds a literal inside those functions again and bypasses the helper, so the
// direct call is the one that keeps the entry point honest.
//
// Error responses are built from kmsg constructors so that Default() supplies
// the values a literal would zero, such as -1 offsets and absent tagged fields.

func TestProduceErrorPartitionsStartFromKmsgDefaults(t *testing.T) {
	resp := &kmsg.ProduceResponse{Version: 9}
	req := &kmsg.ProduceRequest{Version: 9, Topics: []kmsg.ProduceRequestTopic{{
		Topic:      "t",
		Partitions: []kmsg.ProduceRequestTopicPartition{{Partition: 3}},
	}}}
	addErrorForAllPartitions(resp, req, 6)

	got := resp.Topics[0].Partitions[0]
	if got.Partition != 3 || got.ErrorCode != 6 || got.BaseOffset != -1 {
		t.Fatalf("unexpected partition fields: %+v", got)
	}
	if want := kmsg.NewProduceResponseTopicPartitionCurrentLeader(); !reflect.DeepEqual(got.CurrentLeader, want) {
		t.Fatalf("CurrentLeader = %+v, want kmsg default %+v", got.CurrentLeader, want)
	}
}

func TestFetchErrorPartitionsStartFromKmsgDefaults(t *testing.T) {
	resp := &kmsg.FetchResponse{Version: 12}
	req := &kmsg.FetchRequest{Version: 12, Topics: []kmsg.FetchRequestTopic{{
		Topic:      "t",
		Partitions: []kmsg.FetchRequestTopicPartition{{Partition: 3}},
	}}}
	addFetchErrorForAllPartitions(resp, req, 6)

	got := resp.Topics[0].Partitions[0]
	if got.Partition != 3 || got.ErrorCode != 6 {
		t.Fatalf("unexpected partition fields: %+v", got)
	}
	if got.LastStableOffset != -1 || got.LogStartOffset != -1 || got.PreferredReadReplica != -1 {
		t.Fatalf("offsets = last-stable %d, log-start %d, preferred replica %d; want -1",
			got.LastStableOffset, got.LogStartOffset, got.PreferredReadReplica)
	}
	if want := kmsg.NewFetchResponseTopicPartitionDivergingEpoch(); !reflect.DeepEqual(got.DivergingEpoch, want) {
		t.Fatalf("DivergingEpoch = %+v, want %+v", got.DivergingEpoch, want)
	}
	if want := kmsg.NewFetchResponseTopicPartitionCurrentLeader(); !reflect.DeepEqual(got.CurrentLeader, want) {
		t.Fatalf("CurrentLeader = %+v, want %+v", got.CurrentLeader, want)
	}
	if want := kmsg.NewFetchResponseTopicPartitionSnapshotID(); !reflect.DeepEqual(got.SnapshotID, want) {
		t.Fatalf("SnapshotID = %+v, want %+v", got.SnapshotID, want)
	}
}
