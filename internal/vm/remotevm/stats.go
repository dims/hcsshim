//go:build windows

package remotevm

import (
	"context"
	"github.com/Microsoft/hcsshim/internal/runhcs/stats"

	"github.com/Microsoft/hcsshim/internal/vm"
)

func (uvm *utilityVM) Stats(ctx context.Context) (*stats.VirtualMachineStatistics, error) {
	return nil, vm.ErrNotSupported
}
