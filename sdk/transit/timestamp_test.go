package transit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	type fields struct {
		Time time.Time
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "1900-01-01",
			fields:  fields{time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)},
			args:    args{([]byte)("-2208988800000")},
			wantErr: false,
		},
		{
			name:    "1970-01-01",
			fields:  fields{time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
			args:    args{([]byte)("0")},
			wantErr: false,
		},
		{
			name:    "2020-12-31",
			fields:  fields{time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
			args:    args{([]byte)("1609372800000")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Timestamp{}
			if err := tr.UnmarshalJSON(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Timestamp.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tr.Time.Equal(tt.fields.Time) {
				t.Errorf("Timestamp.UnmarshalJSON() = %v, want %v", tr, tt.fields)
			}
		})
	}
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	type fields struct {
		Time time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:    "1900-01-01",
			fields:  fields{time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)},
			want:    ([]byte)(`"-2208988800000"`),
			wantErr: false,
		},
		{
			name:    "1970-01-01",
			fields:  fields{time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
			want:    ([]byte)(`"0"`),
			wantErr: false,
		},
		{
			name:    "2020-12-31",
			fields:  fields{time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
			want:    ([]byte)(`"1609372800000"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := Timestamp{
				Time: tt.fields.Time,
			}
			got, err := tr.MarshalJSON()
			// println(tt.name, tr, got, err, string(got))
			if (err != nil) != tt.wantErr {
				t.Errorf("Timestamp.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Timestamp.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimestampStructMarshalJSON(t *testing.T) {
	value := struct {
		NamedField Timestamp
	}{
		Timestamp{time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
	}
	output, err := json.Marshal(value)
	expected := `{"NamedField":"1609372800000"}`
	if err != nil {
		t.Errorf("json.Marshal returned an error: %v", err)
	}
	if expected != string(output) {
		t.Errorf("json.Marshal returned %v want %v", string(output), expected)
	}
}

func TestTimestampStructUnmarshalJSON(t *testing.T) {
	input := []byte(`{"NamedField":"1609372800000"}`)
	value := struct {
		NamedField Timestamp
	}{}
	expected := struct {
		NamedField Timestamp
	}{
		Timestamp{time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
	}

	err := json.Unmarshal(input, &value)
	if err != nil {
		t.Errorf("json.Unmarshal returned an error: %v", err)
	}
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("json.Unmarshal returned %v, want %v", value, expected)
	}
}

// BenchmarkTimestampMarshallerSprintf benchmarks Sprintf based marshaller
func BenchmarkTimestampMarshallerSprintf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		t1 := time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
		i1 := t1.UnixNano() / int64(time.Millisecond)
		buf := []byte(fmt.Sprintf(`"%d"`, i1))
		if len(buf) == 0 {
			b.Errorf("buf is empty")
		}
	}
}

// BenchmarkTimestampMarshallerAppend benchmarks slice append based marshaller
func BenchmarkTimestampMarshallerAppend(b *testing.B) {
	for i := 0; i < b.N; i++ {
		t1 := time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
		i1 := t1.UnixNano() / int64(time.Millisecond)
		buf := make([]byte, 0, 16)
		buf = append(buf, '"')
		buf = strconv.AppendInt(buf, i1, 10)
		buf = append(buf, '"')
		if len(buf) == 0 {
			b.Errorf("buf is empty")
		}
	}
}
