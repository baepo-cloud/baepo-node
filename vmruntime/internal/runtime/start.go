package runtime

import (
	"context"
)

func (r *Runtime) Start(ctx context.Context) error {
	if err := r.buildInitRamFS(ctx); err != nil {
		return err
	} else if err = r.startHypervisor(ctx); err != nil {
		return err
	} else if err = r.createVM(ctx); err != nil {
		return err
	} else if err = r.bootVM(ctx); err != nil {
		return err
	}

	return nil
}
