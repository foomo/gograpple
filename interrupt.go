package gograpple

import (
	"os"
	"os/signal"
	"time"
)

type Interrupt struct {
	signalChan chan os.Signal
	exitChan   chan bool
	reloadChan chan bool
}

func (g Grapple) registerInterrupt(resetDuration time.Duration) *Interrupt {
	ir := &Interrupt{}
	ir.signalChan = make(chan os.Signal, 1)
	signal.Notify(ir.signalChan)
	ir.exitChan = make(chan bool)
	ir.reloadChan = make(chan bool)
	exiting := false
	readyToReset := false
	i := 0
	go func() {
		for {
			select {
			case <-time.After(resetDuration):
				if readyToReset {
					g.l.Info("resetting termination timer")
					readyToReset = false
				}
				i = 0
			case sig := <-ir.signalChan:
				switch sig {
				case os.Interrupt:
					g.l.Infof("received interrupt signal, trigger one more interrupt within %v to terminate", resetDuration)
					readyToReset = true
					if exiting {
						g.l.Warn("already exiting - ignoring interupt")
						continue
					}
					if i == 0 {
						g.l.Info("triggering reload")
						ir.reloadChan <- true
					} else {
						g.l.Info("triggering exit")
						exiting = true
						ir.exitChan <- true
					}
					i++
				}
			}
		}
	}()
	return ir
}

func (ir *Interrupt) wait(onExit func() error, onLoad func() error) error {
	// initial load
	ir.reloadChan <- true
	// block until an event is triggered
	for {
		select {
		case <-ir.exitChan:
			return onExit()
		default:
		case <-ir.reloadChan:
			if err := onLoad(); err != nil {
				return err
			}
		}
	}
}
