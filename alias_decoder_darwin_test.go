package cocoa

import (
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
				CNID:       0,
			},
			targetPath: "/Users/mattetti/Downloads/3bdc4314e98d2e3a39d9c84443129896f30c2dcf7f99c3aec92f577315916a38.wav",
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
			if got.CNID != tt.want.CNID {
				t.Errorf("AliasFromReader().CNID = %v, want %v", got.CNID, tt.want.CNID)
				return
			}
			if got.TargetPath() != tt.want.TargetPath() {
				t.Errorf("AliasFromReader().TargetPath() = %v, want %v", got.TargetPath(), tt.want.TargetPath())
				return
			}
		})
	}
}
