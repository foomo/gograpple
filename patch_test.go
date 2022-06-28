package gograpple

import (
	"testing"
)

func TestGrapple_Patch(t *testing.T) {
	type args struct {
		repo       string
		dockerfile string
		container  string
		mounts     []Mount
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test", args{"k3d-local-registry:50442", "", "", nil}, false},
	}
	g := testGrapple(t, "example")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := g.Patch(tt.args.repo, tt.args.dockerfile, "", tt.args.container, tt.args.mounts); (err != nil) != tt.wantErr {
				t.Errorf("Grapple.Patch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
