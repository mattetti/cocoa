package cocoa

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestAliasFromReader(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       *BookmarkData
		targetPath string
	}{
		{name: "normal alias",
			input: "fixtures/alias",
			want: &BookmarkData{
				Path:       []string{"Users", "mattetti", "Downloads", "3bdc4314e98d2e3a39d9c84443129896f30c2dcf7f99c3aec92f577315916a38.wav"},
				VolumeURL:  "file:///",
				VolumePath: "/",
				VolumeName: "Macintosh HD",
				CNID:       0,
				VolumeProperties: []uint8{0x81, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0,
					0xef, 0x13, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0,
					0xef, 0x13, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0},
				FileProperties: []uint8{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x1f, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x1f, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				ContainingFolderIDX: 2,
			},
			targetPath: "/Users/mattetti/Downloads/3bdc4314e98d2e3a39d9c84443129896f30c2dcf7f99c3aec92f577315916a38.wav",
		},
		{name: "ExtFat alias",
			input: "fixtures/exFATAlias",
			want: &BookmarkData{
				Path:       []string{"Volumes", "MattSplice", "file.wav"},
				VolumeURL:  "file:///Volumes/MattSplice/",
				VolumePath: "/Volumes/MattSplice",
				VolumeName: "MattSplice",
				CNID:       0,
				VolumeProperties: []uint8{0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0xef, 0x13, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0,
					0xef, 0x13, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0},
				FileProperties: []uint8{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x1f, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
					0x1f, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				ContainingFolderIDX: 0,
			},
			targetPath: "/Volumes/MattSpliceVolumes/MattSplice/file.wav",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.input)
			if err != nil {
				t.Fatal(f)
			}
			defer f.Close()
			got, err := AliasFromReader(f)
			if err != nil {
				t.Errorf("AliasFromReader() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got.Path, tt.want.Path) {
				t.Errorf("AliasFromReader().Path = %v, want %v", got.Path, tt.want.Path)
				return
			}
			if got.VolumeURL != tt.want.VolumeURL {
				t.Errorf("AliasFromReader().VolumeURL = %v, want %v", got.VolumeURL, tt.want.VolumeURL)
				return
			}
			if got.VolumePath != tt.want.VolumePath {
				t.Errorf("AliasFromReader().VolumePath = %v, want %v", got.VolumePath, tt.want.VolumePath)
				return
			}
			if got.VolumeName != tt.want.VolumeName {
				t.Errorf("AliasFromReader().VolumeName = %v, want %v", got.VolumeName, tt.want.VolumeName)
				return
			}
			if got.CNID != tt.want.CNID {
				t.Errorf("AliasFromReader().CNID = %v, want %v", got.CNID, tt.want.CNID)
				return
			}
			if got.ContainingFolderIDX != tt.want.ContainingFolderIDX {
				t.Errorf("AliasFromReader().ContainingFolderIDX = %v, want %v", got.ContainingFolderIDX, tt.want.ContainingFolderIDX)
				return
			}
			if bytes.Compare(got.VolumeProperties, tt.want.VolumeProperties) != 0 {
				t.Errorf("AliasFromReader().VolumeProperties = %#v, want %#v", got.VolumeProperties, tt.want.VolumeProperties)
				return
			}
			if bytes.Compare(got.FileProperties, tt.want.FileProperties) != 0 {
				t.Errorf("AliasFromReader().FileProperties = %#v, want %#v", got.FileProperties, tt.want.FileProperties)
				return
			}
			if got.TargetPath() != tt.targetPath {
				t.Errorf("AliasFromReader().TargetPath() = %v, want %v", got.TargetPath(), tt.targetPath)
				return
			}
		})
	}
}
