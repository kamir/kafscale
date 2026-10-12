package main

import (
	"reflect"
	"testing"

	"github.com/twmb/franz-go/pkg/kmsg"
)

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
