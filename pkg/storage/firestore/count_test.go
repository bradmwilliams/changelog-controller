package firestore

import (
	"strings"
	"testing"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

func TestAggregationCount(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    int64
		wantErr string
	}{
		{
			name:  "native int64",
			value: int64(42),
			want:  42,
		},
		{
			name:  "proto integer value",
			value: &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: 42}},
			want:  42,
		},
		{
			name:  "proto double value",
			value: &pb.Value{ValueType: &pb.Value_DoubleValue{DoubleValue: 42}},
			want:  42,
		},
		{
			name:    "unsupported type",
			value:   "42",
			wantErr: "unexpected type string",
		},
		{
			name:    "proto value without integer or double",
			value:   &pb.Value{ValueType: &pb.Value_StringValue{StringValue: "42"}},
			wantErr: "unexpected type *firestorepb.Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregationCount(tt.value)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("aggregationCount() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("aggregationCount() error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("aggregationCount() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("aggregationCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
