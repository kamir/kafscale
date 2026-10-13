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
	"context"
	"reflect"
	"testing"

	"github.com/twmb/franz-go/pkg/kmsg"
)

// The proxy splits a client request by owning broker and forwards one sub-request per
// broker. The sub-request must be built with the generated constructor: a struct literal
// leaves every unnamed field at the Go zero value, and kmsg writes a tagged field
// whenever it differs from its default.
//
// For Fetch that is ReplicaState, tagged field 1, default {ID: -1, Epoch: -1}. A literal
// leaves {0, 0}, so every flexible sub-request (Fetch v12 and later) told the broker the
// fetch came from replica 0. The Kafka schema defines the field from v15.
//
// Oracle, as in kmsg_defaults_test.go: encode, decode at the same version. ReadFrom
// applies Default() first, so a ReplicaState that comes back different from the default
// proves the tag crossed the wire.

func fetchRequestForSplit(version int16) *kmsg.FetchRequest {
	req := kmsg.NewPtrFetchRequest()
	req.SetVersion(version)
	req.MaxWaitMillis = 500
	req.MinBytes = 1
	req.MaxBytes = 1 << 20
	req.IsolationLevel = 1
	req.SessionID = 0
	req.SessionEpoch = -1
	topic := kmsg.NewFetchRequestTopic()
	topic.Topic = "tsx.wind.series-stats.v2"
	for _, p := range []int32{0, 1} {
		part := kmsg.NewFetchRequestTopicPartition()
		part.Partition = p
		part.FetchOffset = 42
		part.PartitionMaxBytes = 1 << 16
		topic.Partitions = append(topic.Partitions, part)
	}
	req.Topics = append(req.Topics, topic)
	return req
}

func TestFetchSubRequestCarriesNoTaggedFields(t *testing.T) {
	want := kmsg.NewFetchRequest().ReplicaState
	for _, version := range []int16{12, 13} {
		req := fetchRequestForSplit(version)
		p := &proxy{} // no router: every partition lands in the one group ""
		groups := p.groupFetchPartitionsByBroker(context.Background(), req, nil)
		if len(groups) != 1 {
			t.Fatalf("v%d: expected one group, got %d", version, len(groups))
		}
		for _, sub := range groups {
			back := kmsg.NewPtrFetchRequest()
			back.SetVersion(version)
			if err := back.ReadFrom(sub.AppendTo(nil)); err != nil {
				t.Fatalf("v%d: decode the forwarded sub-request: %v", version, err)
			}
			if !reflect.DeepEqual(back.ReplicaState, want) {
				t.Errorf("v%d: ReplicaState crossed the wire as %+v, want the kmsg default %+v",
					version, back.ReplicaState, want)
			}
			if back.ClusterID != nil {
				t.Errorf("v%d: ClusterID crossed the wire as %q, want it absent", version, *back.ClusterID)
			}
		}
	}
}

func TestFetchSubRequestKeepsWhatTheClientAsked(t *testing.T) {
	req := fetchRequestForSplit(13)
	p := &proxy{}
	for _, sub := range p.groupFetchPartitionsByBroker(context.Background(), req, nil) {
		if sub.Version != req.Version || sub.ReplicaID != req.ReplicaID ||
			sub.MaxWaitMillis != req.MaxWaitMillis || sub.MinBytes != req.MinBytes ||
			sub.MaxBytes != req.MaxBytes || sub.IsolationLevel != req.IsolationLevel ||
			sub.SessionID != req.SessionID || sub.SessionEpoch != req.SessionEpoch {
			t.Errorf("sub-request header differs from the client request:\n sub %+v\n req %+v", sub, req)
		}
		if len(sub.Topics) != 1 || len(sub.Topics[0].Partitions) != 2 {
			t.Fatalf("sub-request lost topics or partitions: %+v", sub.Topics)
		}
		if !reflect.DeepEqual(sub.Topics[0].Partitions, req.Topics[0].Partitions) {
			t.Errorf("partitions changed on the way:\n sub %+v\n req %+v", sub.Topics[0].Partitions, req.Topics[0].Partitions)
		}
	}
}

func TestProduceSubRequestKeepsWhatTheClientAsked(t *testing.T) {
	const version int16 = 9
	txn := "txn-1"
	req := kmsg.NewPtrProduceRequest()
	req.SetVersion(version)
	req.Acks = -1
	req.TimeoutMillis = 4321
	req.TransactionID = &txn
	topic := kmsg.NewProduceRequestTopic()
	topic.Topic = "tsx.wind.series-stats.v2"
	part := kmsg.NewProduceRequestTopicPartition()
	part.Partition = 3
	topic.Partitions = append(topic.Partitions, part)
	req.Topics = append(req.Topics, topic)

	p := &proxy{}
	groups := p.groupPartitionsByBroker(context.Background(), req, nil)
	if len(groups) != 1 {
		t.Fatalf("expected one group, got %d", len(groups))
	}
	for _, sub := range groups {
		back := kmsg.NewPtrProduceRequest()
		back.SetVersion(version)
		if err := back.ReadFrom(sub.AppendTo(nil)); err != nil {
			t.Fatalf("decode the forwarded sub-request: %v", err)
		}
		if back.Acks != req.Acks || back.TimeoutMillis != req.TimeoutMillis ||
			back.TransactionID == nil || *back.TransactionID != txn {
			t.Errorf("sub-request header differs from the client request: acks %d timeout %d txn %v",
				back.Acks, back.TimeoutMillis, back.TransactionID)
		}
		if len(back.Topics) != 1 || len(back.Topics[0].Partitions) != 1 || back.Topics[0].Partitions[0].Partition != 3 {
			t.Errorf("sub-request lost its partition: %+v", back.Topics)
		}
	}
}
