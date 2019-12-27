package subseconds

import (
	"reflect"
	"testing"
	"time"
)

func TestMillisecondTimestamp_UnmarshalJSON(t *testing.T) {
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
			tr := &MillisecondTimestamp{}
			if err := tr.UnmarshalJSON(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("MillisecondTimestamp.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tr.Time.Equal(tt.fields.Time) {
				t.Errorf("MillisecondTimestamp.UnmarshalJSON() = %v, want %v", tr, tt.fields)
			}
		})
	}
}

func TestMillisecondTimestamp_MarshalJSON(t *testing.T) {
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
			want:    ([]byte)("\"-2208988800000\""),
			wantErr: false,
		},
		{
			name:    "1970-01-01",
			fields:  fields{time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
			want:    ([]byte)("\"0\""),
			wantErr: false,
		},
		{
			name:    "2020-12-31",
			fields:  fields{time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
			want:    ([]byte)("\"1609372800000\""),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := MillisecondTimestamp{
				Time: tt.fields.Time,
			}
			got, err := tr.MarshalJSON()
			// log.Println(tt.name, tr, got, err, string(got))
			if (err != nil) != tt.wantErr {
				t.Errorf("MillisecondTimestamp.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MillisecondTimestamp.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
