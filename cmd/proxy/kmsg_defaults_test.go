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
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/KafScale/platform/pkg/protocol"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Response structs must be created with the generated New...() constructors, which apply
// Default(). A Go struct literal leaves every field at the zero value, and kmsg decides
// whether to emit a tagged field by comparing it against its default. A zero-valued
// tagged field therefore counts as "set" and is written to the wire.
//
// For Produce that is fatal: the tagged field CurrentLeader is tag 0, kmsg gates it on
// flexibility only (Produce is flexible from v9) while the Kafka schema defines it from
// v10. A v9 response carrying tag 0 makes the Apache Kafka Java client throw
// "Tag 0 is not valid for version 9" on the producer I/O thread, which is not
// recoverable, so the client dies.
//
// Oracle: encode at v9, decode at v9. readFrom applies Default() per element, so a
// CurrentLeader that comes back different from the default proves the tag crossed the wire.

func TestProduceErrorResponseCarriesNoTaggedFieldsAtV9(t *testing.T) {
	const version int16 = 9

	req := kmsg.NewPtrProduceRequest()
	req.SetVersion(version)
	topic := kmsg.NewProduceRequestTopic()
	topic.Topic = "tsx.wind.series-stats.v2"
	for _, p := range []int32{0, 1, 2} {
		rp := kmsg.NewProduceRequestTopicPartition()
		rp.Partition = p
		topic.Partitions = append(topic.Partitions, rp)
	}
	req.Topics = append(req.Topics, topic)

	resp := kmsg.NewPtrProduceResponse()
	resp.SetVersion(version)
	addErrorForAllPartitions(resp, req, protocol.REQUEST_TIMED_OUT)

	wire := protocol.EncodeResponse(7, version, resp)
	body, ok := protocol.SkipResponseHeader(protocol.APIKeyProduce, version, wire)
	if !ok {
		t.Fatal("could not skip the response header")
	}

	decoded := kmsg.NewPtrProduceResponse()
	decoded.SetVersion(version)
	if err := decoded.ReadFrom(body); err != nil {
		t.Fatalf("decode v%d produce response: %v", version, err)
	}

	want := kmsg.NewProduceResponseTopicPartition().CurrentLeader
	parts := 0
	for _, tp := range decoded.Topics {
		for _, p := range tp.Partitions {
			parts++
			if !reflect.DeepEqual(p.CurrentLeader, want) {
				t.Errorf("topic %q partition %d: CurrentLeader crossed the wire as %+v, want the kmsg default %+v; "+
					"tagged field 0 in a v9 ProduceResponse kills the Kafka Java client "+
					"(\"Tag 0 is not valid for version 9\")", tp.Topic, p.Partition, p.CurrentLeader, want)
			}
			unknown := 0
			p.UnknownTags.Each(func(uint32, []byte) { unknown++ })
			if unknown != 0 {
				t.Errorf("topic %q partition %d: %d unknown tagged field(s) on the wire, want 0", tp.Topic, p.Partition, unknown)
			}
		}
	}
	if parts != 3 {
		t.Fatalf("decoded %d partitions, want 3", parts)
	}
}

func TestFetchErrorPartitionKeepsKmsgDefaults(t *testing.T) {
	// FetchResponseTopicPartition has three tagged fields (DivergingEpoch, CurrentLeader,
	// SnapshotID) plus three scalars that default to -1. A literal silently turns
	// PreferredReadReplica into 0, which tells the client to fetch from broker 0.
	def := kmsg.NewFetchResponseTopicPartition()

	got := newFetchErrorPartition(11, protocol.REQUEST_TIMED_OUT)

	if !reflect.DeepEqual(got.DivergingEpoch, def.DivergingEpoch) {
		t.Errorf("DivergingEpoch = %+v, want the default %+v", got.DivergingEpoch, def.DivergingEpoch)
	}
	if !reflect.DeepEqual(got.CurrentLeader, def.CurrentLeader) {
		t.Errorf("CurrentLeader = %+v, want the default %+v", got.CurrentLeader, def.CurrentLeader)
	}
	if !reflect.DeepEqual(got.SnapshotID, def.SnapshotID) {
		t.Errorf("SnapshotID = %+v, want the default %+v", got.SnapshotID, def.SnapshotID)
	}
	if got.PreferredReadReplica != def.PreferredReadReplica {
		t.Errorf("PreferredReadReplica = %d, want the default %d (0 means \"fetch from broker 0\")",
			got.PreferredReadReplica, def.PreferredReadReplica)
	}
	if got.LastStableOffset != def.LastStableOffset {
		t.Errorf("LastStableOffset = %d, want the default %d", got.LastStableOffset, def.LastStableOffset)
	}
	if got.LogStartOffset != def.LogStartOffset {
		t.Errorf("LogStartOffset = %d, want the default %d", got.LogStartOffset, def.LogStartOffset)
	}
}

// TestNoKmsgStructLiteralsWithDefaults keeps the bug from coming back, on both sides of
// the wire. A kmsg request or response struct whose Default() is non-empty must not be
// built as a literal. Checked by source scan, because the defect is invisible at compile
// time and only shows on the wire.
//
// Two forms are caught: the plain literal `kmsg.X{...}`, and a slice literal with elided
// element types, `[]kmsg.X{{...}}`, which builds the same zero-valued elements. A slice
// of values that were constructed elsewhere, `[]kmsg.X{a, b}`, is fine and not flagged.
func TestNoKmsgStructLiteralsWithDefaults(t *testing.T) {
	// Structs whose Default() is empty; a literal cannot lose anything there. Each entry
	// carries an instance so the claim is checked below instead of trusted.
	emptyDefaults := map[string]interface{ Default() }{
		"ProduceResponse":              &kmsg.ProduceResponse{},
		"ProduceResponseTopic":         &kmsg.ProduceResponseTopic{},
		"FetchResponse":                &kmsg.FetchResponse{},
		"FetchResponseTopic":           &kmsg.FetchResponseTopic{},
		"ApiVersionsResponseApiKey":    &kmsg.ApiVersionsResponseApiKey{},
		"JoinGroupResponseMember":      &kmsg.JoinGroupResponseMember{},
		"ProduceRequestTopic":          &kmsg.ProduceRequestTopic{},
		"ProduceRequestTopicPartition": &kmsg.ProduceRequestTopicPartition{},
		"FetchRequestTopic":            &kmsg.FetchRequestTopic{},
	}
	// The allow-list must stay true. A kmsg upgrade that gives one of these structs a
	// default would otherwise hide behind the list.
	for name, zero := range emptyDefaults {
		typ := reflect.TypeOf(zero).Elem()
		defaulted := reflect.New(typ)
		defaulted.Interface().(interface{ Default() }).Default()
		if !reflect.DeepEqual(reflect.Zero(typ).Interface(), defaulted.Elem().Interface()) {
			t.Errorf("kmsg.%s is listed as having an empty Default(), but Default() changes it to %+v; "+
				"take it off the list and build it with kmsg.New%s()", name, defaulted.Elem().Interface(), name)
		}
	}

	const name = `([A-Z][A-Za-z]*(?:Request|Response)[A-Za-z]*)`
	plain := regexp.MustCompile(`(?:^|[^\]])kmsg\.` + name + `\{`)
	elided := regexp.MustCompile(`\[\]kmsg\.` + name + `\{\s*\{`)

	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(b), "\n") {
			var found []string
			for _, m := range plain.FindAllStringSubmatch(line, -1) {
				found = append(found, m[1])
			}
			for _, m := range elided.FindAllStringSubmatch(line, -1) {
				found = append(found, m[1])
			}
			for _, name := range found {
				if _, ok := emptyDefaults[name]; ok {
					continue
				}
				t.Errorf("%s:%d builds kmsg.%s as a struct literal, so Default() never runs; "+
					"use kmsg.New%s() and assign the fields", path, i+1, name, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
