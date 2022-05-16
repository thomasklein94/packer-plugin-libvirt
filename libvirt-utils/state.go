package libvirtutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/digitalocean/go-libvirt"
)

func DomainStateMeansStopped(state libvirt.DomainState) (res bool) {
	res = false

	switch state {
	case libvirt.DomainShutoff, libvirt.DomainCrashed:
		res = true
	}

	return
}

func PollDomainState(
	ctx context.Context,
	period time.Duration,
	driver libvirt.Libvirt,
	domain libvirt.Domain,
	result chan<- libvirt.DomainState,
	errs chan<- error,
) {

	for {
		state_i32, reason, err := driver.DomainGetState(domain, 0)
		state := libvirt.DomainState(state_i32)

		log.Printf("DomainGetState.Results: State(%d) Reason(%d) Error(%s)\n", state, reason, err)

		if err != nil {
			err = fmt.Errorf("DomainGetState.RPC: %s", err)
			errs <- err
			return
		}

		result <- state

		select {
		case <-ctx.Done():
			err = ctx.Err()
			log.Printf("domain_shutdown: %s", err)
			return
		case <-time.After(period):
		}
	}
}
