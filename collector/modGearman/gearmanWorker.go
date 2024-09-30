package modGearman

import (
	"time"

	"github.com/fvrflho/nagflux/collector"
	"github.com/fvrflho/nagflux/collector/livestatus"
	"github.com/fvrflho/nagflux/collector/spoolfile"
	"github.com/fvrflho/nagflux/config"
	"github.com/fvrflho/nagflux/helper"
	"github.com/fvrflho/nagflux/helper/crypto"
	"github.com/fvrflho/nagflux/logging"
	libworker "github.com/appscode/g2/worker"
	"github.com/kdar/factorlog"
)

// GearmanWorker queries the gearmanserver and adds the extraced perfdata to the queue.
type GearmanWorker struct {
	runQuit               chan bool
	loadQuit              chan bool
	pauseQuit             chan bool
	results               collector.ResultQueues
	nagiosSpoolfileWorker *spoolfile.NagiosSpoolfileWorker
	aesECBDecrypter       *crypto.AESECBDecrypter
	worker                *libworker.Worker
	log                   *factorlog.FactorLog
	jobQueue              string
	address               string
}

// NewGearmanWorker generates a new GearmanWorker.
// leave the key empty to disable encryption, otherwise the gearmanpacketes are expected to be encrpyten with AES-ECB 128Bit and a 32 Byte Key.
func NewGearmanWorker(address, queue, key string, results collector.ResultQueues, livestatusCacheBuilder *livestatus.CacheBuilder) *GearmanWorker {
	var decrypter *crypto.AESECBDecrypter
	if key != "" {
		byteKey := ShapeKey(key, DefaultModGearmanKeyLength)
		var err error
		decrypter, err = crypto.NewAESECBDecrypter(byteKey)
		if err != nil {
			panic(err)
		}
	}
	worker := &GearmanWorker{
		runQuit:   make(chan bool, 1),
		loadQuit:  make(chan bool, 1),
		pauseQuit: make(chan bool, 1),
		results:   results,
		nagiosSpoolfileWorker: spoolfile.NewNagiosSpoolfileWorker(
			-1, make(chan string), make(collector.ResultQueues), livestatusCacheBuilder, 4096, collector.AllFilterable,
		),
		aesECBDecrypter: decrypter,
		worker:          nil,
		address:         address,
		log:             logging.GetLogger(),
		jobQueue:        queue,
	}
	go worker.run()
	go worker.handleLoad()
	go worker.handlePause()

	return worker
}

func (g *GearmanWorker) startGearmanWorker() error {
	g.shutdownGearmanWorker()
	g.worker = libworker.New(libworker.OneByOne)
	g.worker.AddServer("tcp4", g.address)
	g.worker.ErrorHandler = func(err error) {
		switch err.(type) {
		case *libworker.WorkerDisconnectError:
			g.log.Warn("Gearmand did not response. Connection closed")
		default:
			g.log.Warn(err)
		}
		g.shutdownGearmanWorker()
	}
	g.worker.AddFunc(g.jobQueue, g.handelJob, libworker.Unlimited)
	if err := g.worker.Ready(); err != nil {
		g.worker = nil
		return err
	}
	g.log.Info("Gearman worker ready")
	go g.worker.Work()
	return nil
}

// shutdown gearman worker
func (g *GearmanWorker) shutdownGearmanWorker() {
	if g.worker == nil {
		return
	}
	g.log.Warnf("shutting down worker")
	g.worker.ErrorHandler = nil
	g.worker.Shutdown()
	g.worker.Close()
	g.worker = nil
}

// Stop stops the worker
func (g *GearmanWorker) Stop() {
	g.shutdownGearmanWorker()
	g.runQuit <- true
	g.loadQuit <- true
	g.pauseQuit <- true
	logging.GetLogger().Debug("GearmanWorker stopped")
}

func (g *GearmanWorker) run() {
	for {
		if g.worker == nil {
			err := g.startGearmanWorker()
			if err != nil {
				g.log.Warn(err)
			}
		}

		// interruptable retry in 10 seconds
		select {
		case <-g.runQuit:
			g.shutdownGearmanWorker()
			return
		case <-time.After(time.Duration(10) * time.Second):
			// retry connection after 10 seconds
		}
	}
}

func (g *GearmanWorker) handleLoad() {
	bufferLimit := int(float32(config.GetConfig().Main.BufferSize) * 0.90)
	for {
		for _, r := range g.results {
			if len(r) > bufferLimit && g.worker != nil {
				g.worker.Lock()
				for len(r) > bufferLimit {
					time.Sleep(time.Duration(100) * time.Millisecond)
				}
				g.worker.Unlock()
			}
		}
		select {
		case <-g.loadQuit:
			return
		case <-time.After(time.Duration(1) * time.Second):
		}
	}
}

func (g *GearmanWorker) handlePause() {
	pause := false
	for {
		select {
		case <-g.pauseQuit:
			return
		case <-time.After(time.Duration(1) * time.Second):
			globalPause := config.IsAnyTargetOnPause()
			if pause != globalPause && g.worker != nil {
				if globalPause {
					g.worker.Lock()
				} else {
					g.worker.Unlock()
				}
				pause = globalPause
			}
		}
	}
}

func (g *GearmanWorker) handelJob(job libworker.Job) ([]byte, error) {
	secret := job.Data()
	if g.aesECBDecrypter != nil {
		var err error
		secret, err = g.aesECBDecrypter.Decypt(secret)
		if err != nil {
			g.log.Warn(err, ". Data: ", string(job.Data()))
			return job.Data(), nil
		}
	}
	splittedPerformanceData := helper.StringToMap(string(secret), "\t", "::")
	g.log.Debug("[ModGearman] ", string(job.Data()))
	g.log.Debug("[ModGearman] ", splittedPerformanceData)
	for singlePerfdata := range g.nagiosSpoolfileWorker.PerformanceDataIterator(splittedPerformanceData) {
		for _, r := range g.results {
			select {
			case r <- singlePerfdata:
			case <-time.After(time.Duration(1) * time.Minute):
				logging.GetLogger().Warn("GearmanWorker: Could not write to buffer")
			}
		}
	}
	return job.Data(), nil
}
