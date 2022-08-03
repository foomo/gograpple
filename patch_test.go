package gograpple

import (
	"testing"
)

func TestGrapple_Patch(t *testing.T) {
	type args struct {
		image     string
		tag       string
		container string
		mounts    []Mount
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test", args{"golang", "latest", "", nil}, false},
	}
	g := testGrapple(t, "example")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := g.Patch(tt.args.image, tt.args.tag, tt.args.container, tt.args.mounts); (err != nil) != tt.wantErr {
				t.Errorf("Grapple.Patch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
