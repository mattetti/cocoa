package cocoa

import (
	"bytes"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestBookmarkData_Write(t *testing.T) {
	tests := []struct {
		name string
		data *BookmarkData
	}{
		{name: "round trip",
			data: &BookmarkData{
				FileSystemType:      "",
				Path:                []string{"Users", "mattetti", "Splice", "sounds", "drums", "727 Maracas.wav"},
				CNIDPath:            []uint32{0x669dc, 0x9b7c3, 0x2c2de1, 0x7f1e94, 0x8a2402, 0x8a2406},
				FileCreationDate:    time.Unix(63190694952, 0),
				FileProperties:      []uint8{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				ContainingFolderIDX: 0x7,
				VolumePath:          "/",
				VolumeIsRoot:        true,
				VolumeURL:           "file:///",
				VolumeName:          "Macintosh HD",
				VolumeSize:          42,
				VolumeCreationDate:  time.Unix(0, 0),
				VolumeUUID:          "",
				VolumeProperties:    []uint8{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				CreationOptions:     0x400,
				WasFileReference:    true,
				UserName:            "mattetti",
				CNID:                0x8b4160,
				UID:                 0x9942,
				Filename:            "727 Maracas.wav",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.data.Write(w); err != nil {
				t.Errorf("BookmarkData.Write() error = %v", err)
				return
			}
			r := bytes.NewReader(w.Bytes())
			got, err := AliasFromReader(r)
			if err != nil {
				f, cerr := os.Create("fixtures/failedTest.hex")
				if cerr != nil {
					t.Fatal(cerr)
				}
				f.Write(w.Bytes())
				f.Close()
				t.Log("Saved failed generated alias to fixtures/failedTest.hex")
				t.Fatal(err)
			}
			if reflect.DeepEqual(got, tt.data) {
				t.Errorf("BookmarkData didn't round trip, expected %v, got %v", tt.data, got)
			}
		})
	}
}
