package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	otlpcommonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/proto"
)

func TestPatchedLogValuesPreserveOTLPRepresentations(t *testing.T) {
	values := []attribute.Value{
		attribute.BoolValue(true), attribute.Int64Value(-5), attribute.Float64Value(1.25),
		attribute.StringValue("payload"), attribute.ByteSliceValue([]byte{0, 255}),
		attribute.SliceValue(attribute.StringValue("nested"), attribute.Int64Value(3)),
		attribute.MapValue(attribute.KeyValue{Key: "nested", Value: attribute.ByteSliceValue([]byte{1, 2})}),
		attribute.BoolSliceValue([]bool{true, false}), attribute.Int64SliceValue([]int64{1, -2}),
		attribute.Float64SliceValue([]float64{1.2, 3.4}), attribute.StringSliceValue([]string{"a", "b"}),
	}
	for _, value := range values {
		t.Run(value.Type().String(), func(t *testing.T) {
			wire := LogValueToPB(value)
			if wire.GetStringValue() == "INVALID" {
				t.Fatal("supported value lost during conversion")
			}
			if got := LogValueToPB(LogValueFromPB(wire)); !proto.Equal(got, wire) {
				t.Fatalf("OTLP roundtrip changed value: got %v, want %v", got, wire)
			}
		})
	}
	key := &otlpcommonv1.KeyValue{Key: "custom.key", Value: LogValueToPB(attribute.StringValue("value"))}
	if got := LogKeyValuesFromPB([]*otlpcommonv1.KeyValue{key}); len(got) != 1 || string(got[0].Key) != key.Key || got[0].Value.AsString() != "value" {
		t.Fatalf("attribute key/value changed: %v", got)
	}
}

func TestPatchedStdioPreservesPayloadAndEOF(t *testing.T) {
	ctx := context.Background()
	processor := &collectLogProcessor{}
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
	ctx = WithLoggerProvider(ctx, provider)
	streams := SpanStdio(ctx, "consumer", attribute.String("custom", "retained"))
	if _, err := streams.Stdout.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := streams.Stderr.Write([]byte("error")); err != nil {
		t.Fatal(err)
	}
	if err := streams.Close(); err != nil {
		t.Fatal(err)
	}
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if len(processor.logs) != 4 {
		t.Fatalf("got %d records, want stdout/stderr payload and both EOF records", len(processor.logs))
	}
	for i, record := range processor.logs {
		attrs := map[attribute.Key]attribute.Value{}
		record.WalkAttributes(func(kv attribute.KeyValue) bool { attrs[kv.Key] = kv.Value; return true })
		if attrs["custom"].AsString() != "retained" || attrs[StdioStreamAttr].AsInt64() != int64(i%2+1) {
			t.Fatalf("record %d lost attributes: %v", i, attrs)
		}
		if i < 2 {
			want := []string{"hello", "error"}[i]
			if record.Body().AsString() != want || attrs[StdioEOFAttr].AsBool() {
				t.Fatalf("record %d changed payload or EOF", i)
			}
		} else if record.Body().AsString() != "" || !attrs[StdioEOFAttr].AsBool() {
			t.Fatalf("record %d lost EOF", i)
		}
	}
}
