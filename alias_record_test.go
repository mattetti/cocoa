package cocoa

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAliasRecord_Encode(t *testing.T) {
	tests := []struct {
		name   string
		record *AliasRecord
		want   string
	}{
		{"real example",
			&AliasRecord{
				Path:             "/Users/mattetti/Code/golang/src/github.com/mattetti/cocoa/cocoa.go",
				CNIDPath:         []uint32{0x669dc, 0x9b7c3, 0x105f25, 0x12fe65, 0x13053d, 0x1f86ca, 0x1fe5c4, 0x7dc0f5},
				PathItems:        []string{"Users", "mattetti", "Code", "golang", "src", "github.com", "mattetti", "cocoa", "cocoa.go"},
				Kind:             0x0,
				VolumeName:       "Macintosh HD",
				VolumeDate:       time.Unix(63629270897, 0),
				FileSystem:       "H+",
				DiskType:         0x0,
				FolderCNID:       0x1fe5c4,
				TargetName:       "cocoa.go",
				TargetCNID:       0x7dc0f5,
				TargetCreation:   time.Unix(63639891333, 0),
				DirsAliasToRoot:  -1,
				DirsRootToTarget: -1,
				VolumeID:         0x0},
			"cocoa.hex",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.record.Encode()
			if err != nil {
				t.Errorf("AliasRecord.Encode() error = %v", err)
				return
			}
			f, err := os.Open(filepath.Join("testExpectations", tt.want))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			for i, b := range raw {
				if got[i] != b {
					t.Errorf("AliasRecord.Encode() = byte at position %d is %#x, expected %#x\n",
						i, got[i], b)
				}
			}
		})
	}
}

func Test_aliasRecordEncoder_encode(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			got, err := e.encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("aliasRecordEncoder.encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("aliasRecordEncoder.encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_aliasRecordEncoder_write(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.write(tt.args.data)
		})
	}
}

func Test_aliasRecordEncoder_add(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		src interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.add(tt.args.src)
		})
	}
}

func Test_aliasRecordEncoder_pascalString(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		str  string
		size int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			if got := e.pascalString(tt.args.str, tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("aliasRecordEncoder.pascalString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_aliasRecordEncoder_dateInSecs(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		t time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint32
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			if got := e.dateInSecs(tt.args.t); got != tt.want {
				t.Errorf("aliasRecordEncoder.dateInSecs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_aliasRecordEncoder_folderName(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			if got := e.folderName(); got != tt.want {
				t.Errorf("aliasRecordEncoder.folderName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_aliasRecordEncoder_folderNameTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.folderNameTag()
		})
	}
}

func Test_aliasRecordEncoder_carbonPathTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.carbonPathTag()
		})
	}
}

func Test_aliasRecordEncoder_posixPathTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.posixPathTag()
		})
	}
}

func Test_aliasRecordEncoder_filenameTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.filenameTag()
		})
	}
}

func Test_aliasRecordEncoder_volumeNameTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.volumeNameTag()
		})
	}
}

func Test_aliasRecordEncoder_dateTags(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.dateTags()
		})
	}
}

func Test_aliasRecordEncoder_cnidPathTag(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			e.cnidPathTag()
		})
	}
}

func Test_aliasRecordEncoder_carbonize(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		str string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			if got := e.carbonize(tt.args.str); got != tt.want {
				t.Errorf("aliasRecordEncoder.carbonize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_aliasRecordEncoder_setError(t *testing.T) {
	type fields struct {
		record *AliasRecord
		buf    *bytes.Buffer
		err    error
	}
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &aliasRecordEncoder{
				record: tt.fields.record,
				buf:    tt.fields.buf,
				err:    tt.fields.err,
			}
			if err := e.setError(tt.args.err); (err != nil) != tt.wantErr {
				t.Errorf("aliasRecordEncoder.setError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
