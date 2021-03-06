package logging

import (
	"github.com/google/martian/log"
	"github.com/jitsucom/eventnative/safego"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"
)

const (
	logFileMaxSizeMB         = 100
	twentyFourHoursInMinutes = 1440
)

//regex for reading already rotated and closed log files
var TokenIdExtractRegexp = regexp.MustCompile("incoming.tok=(.*)-\\d\\d\\d\\d-\\d\\d-\\d\\dT")

//RollingWriterProxy for lumberjack.Logger
//Rotate() only if file isn't empty
type RollingWriterProxy struct {
	lWriter       *lumberjack.Logger
	rotateOnClose bool

	records uint64
}

func NewRollingWriter(config Config) io.WriteCloser {
	fileNamePath := filepath.Join(config.FileDir, config.FileName+".log")
	lWriter := &lumberjack.Logger{
		Filename: fileNamePath,
		MaxSize:  logFileMaxSizeMB,
		Compress: config.Compress,
	}
	if config.MaxBackups > 0 {
		lWriter.MaxBackups = config.MaxBackups
	}

	rwp := &RollingWriterProxy{lWriter: lWriter, records: 0, rotateOnClose: config.RotateOnClose}

	if config.RotationMin == 0 {
		config.RotationMin = twentyFourHoursInMinutes
	}
	rotation := time.Duration(config.RotationMin) * time.Minute

	ticker := time.NewTicker(rotation)
	safego.RunWithRestart(func() {
		for {
			<-ticker.C
			if atomic.SwapUint64(&rwp.records, 0) > 0 {
				if err := lWriter.Rotate(); err != nil {
					log.Errorf("Error rotating log file [%s]: %v", config.FileName, err)
				}
			}
		}
	})

	return rwp
}

func (rwp *RollingWriterProxy) Write(p []byte) (int, error) {
	atomic.AddUint64(&rwp.records, 1)
	return rwp.lWriter.Write(p)
}

func (rwp *RollingWriterProxy) Close() error {
	if rwp.rotateOnClose {
		if err := rwp.lWriter.Rotate(); err != nil {
			log.Errorf("Error rotating log file: %v", err)
		}
	}

	return rwp.lWriter.Close()
}
