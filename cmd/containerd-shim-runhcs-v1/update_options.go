package main

import (
	runhcsopts "github.com/Microsoft/hcsshim/internal/runhcs/options"
	"github.com/Microsoft/hcsshim/pkg/annotations"
	"github.com/opencontainers/runtime-spec/specs-go"

	"strconv"
)

// UpdateSpecFromOptions sets extra annotations on the OCI spec based on the
// `opts` struct.
func UpdateSpecFromOptions(s specs.Spec, opts *runhcsopts.Options) specs.Spec {
	if opts == nil {
		return s
	}

	if _, ok := s.Annotations[annotations.BootFilesRootPath]; !ok && opts.BootFilesRootPath != "" {
		s.Annotations[annotations.BootFilesRootPath] = opts.BootFilesRootPath
	}

	if _, ok := s.Annotations[annotations.ProcessorCount]; !ok && opts.VmProcessorCount != 0 {
		s.Annotations[annotations.ProcessorCount] = strconv.FormatInt(int64(opts.VmProcessorCount), 10)
	}

	if _, ok := s.Annotations[annotations.MemorySizeInMB]; !ok && opts.VmMemorySizeInMb != 0 {
		s.Annotations[annotations.MemorySizeInMB] = strconv.FormatInt(int64(opts.VmMemorySizeInMb), 10)
	}

	if _, ok := s.Annotations[annotations.NetworkConfigProxy]; !ok && opts.NCProxyAddr != "" {
		s.Annotations[annotations.NetworkConfigProxy] = opts.NCProxyAddr
	}

	for key, value := range opts.DefaultContainerAnnotations {
		// Make sure not to override any annotations which are set explicitly
		if _, ok := s.Annotations[key]; !ok {
			s.Annotations[key] = value
		}
	}

	return s
}
