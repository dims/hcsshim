//go:build windows

package oci

import (
	"context"
	"fmt"
	"testing"

	"github.com/Microsoft/hcsshim/internal/uvm"
	"github.com/Microsoft/hcsshim/pkg/annotations"
	"github.com/google/go-cmp/cmp"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Test_SpecToUVMCreateOptions_Default_LCOW(t *testing.T) {
	s := &specs.Spec{
		Linux:       &specs.Linux{},
		Annotations: make(map[string]string),
	}

	opts, err := SpecToUVMCreateOpts(context.Background(), s, t.Name(), "")
	if err != nil {
		t.Fatalf("could not generate creation options from spec: %v", err)
	}

	lopts := (opts).(*uvm.OptionsLCOW)
	dopts := uvm.NewDefaultOptionsLCOW(t.Name(), "")

	// output handler equality is always false, so set to nil
	lopts.OutputHandlerCreator = nil
	dopts.OutputHandlerCreator = nil

	if !cmp.Equal(*lopts, *dopts) {
		t.Fatalf("should not have updated create options from default when no annotation are provided:\n%s", cmp.Diff(lopts, dopts))
	}
}

func Test_SpecToUVMCreateOptions_Default_WCOW(t *testing.T) {
	s := &specs.Spec{
		Windows: &specs.Windows{
			HyperV: &specs.WindowsHyperV{},
		},
		Annotations: make(map[string]string),
	}

	opts, err := SpecToUVMCreateOpts(context.Background(), s, t.Name(), "")
	if err != nil {
		t.Fatalf("could not generate creation options from spec: %v", err)
	}

	wopts := (opts).(*uvm.OptionsWCOW)
	dopts := uvm.NewDefaultOptionsWCOW(t.Name(), "")

	if !cmp.Equal(*wopts, *dopts) {
		t.Fatalf("should not have updated create options from default when no annotation are provided:\n%s", cmp.Diff(wopts, dopts))
	}
}

func Test_SpecToUVMCreateOptions_Common(t *testing.T) {
	cpugroupid := "1"
	lowmmiogap := 1024
	as := map[string]string{
		annotations.ProcessorCount:            "8",
		annotations.CPUGroupID:                cpugroupid,
		annotations.DisableWritableFileShares: "true",
		annotations.MemoryLowMMIOGapInMB:      fmt.Sprint(lowmmiogap),
	}

	tests := []struct {
		name    string
		spec    specs.Spec
		extract func(interface{}) *uvm.Options
	}{
		{
			name: "lcow",
			spec: specs.Spec{
				Linux: &specs.Linux{},
			},
			// generics would be nice ...
			extract: func(i interface{}) *uvm.Options {
				o := (i).(*uvm.OptionsLCOW)
				return o.Options
			},
		},
		{
			name: "wcow-hypervisor",
			spec: specs.Spec{
				Windows: &specs.Windows{
					HyperV: &specs.WindowsHyperV{},
				},
			},
			extract: func(i interface{}) *uvm.Options {
				o := (i).(*uvm.OptionsWCOW)
				return o.Options
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.spec.Annotations = as
			opts, err := SpecToUVMCreateOpts(context.Background(), &tt.spec, t.Name(), "")
			if err != nil {
				t.Fatalf("could not generate creation options from spec: %v", err)
			}

			// get the underlying uvm.Options from uvm.Options[LW]COW
			copts := tt.extract(opts)
			if copts.LowMMIOGapInMB != uint64(lowmmiogap) {
				t.Fatalf("should have updated creation options low MMIO Gap when annotation is provided: %v != %v", copts.LowMMIOGapInMB, lowmmiogap)
			}
			if copts.ProcessorCount != 8 {
				t.Fatalf("should have updated creation options processor count when annotation is provided: %v != 8", copts.ProcessorCount)
			}
			if copts.CPUGroupID != cpugroupid {
				t.Fatalf("should have updated creation options CPU group Id when annotation is provided: %v != %v", copts.CPUGroupID, cpugroupid)
			}
			if !copts.NoWritableFileShares {
				t.Fatal("should have disabled writable in shares creation when annotation is provided")
			}
		})
	}
}
