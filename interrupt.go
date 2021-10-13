package gograpple

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
)

func RunWithInterrupt(l *logrus.Entry, callback func(ctx context.Context)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	durReload := 3 * time.Second
	for {
		ctx, cancelCtx := context.WithCancel(context.Background())
		// do stuff
		go callback(ctx)
		select {
		case <-signalChan: // first signal
			l.Info("-")
			l.Infof("interrupt received, trigger one more within %v to terminate", durReload)
			cancelCtx()
			select {
			case <-time.After(durReload): // reloads durReload after first signal
				l.Info("-")
				l.Info("reloading")
			case <-signalChan: // second signal, hard exit
				l.Info("-")
				l.Info("terminating")
				signal.Stop(signalChan)
				// exit loop
				return
			}
		}
	}
}
