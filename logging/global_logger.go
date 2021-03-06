package logging

import (
	"errors"
	"fmt"
	"github.com/gookit/color"
	"github.com/jitsucom/eventnative/notifications"
	"io"
	"log"
	"strings"
)

const (
	errPrefix   = "[ERROR]:"
	warnPrefix  = "[WARN]:"
	infoPrefix  = "[INFO]:"
	debugPrefix = "[DEBUG]:"
)

var GlobalLogsWriter io.Writer
var ConfigWarn string

type Config struct {
	FileName    string
	FileDir     string
	RotationMin int64
	MaxBackups  int
	Compress    bool

	RotateOnClose bool
}

func (c Config) Validate() error {
	if c.FileName == "" {
		return errors.New("Logger file name can't be empty")
	}
	if c.FileDir == "" {
		return errors.New("Logger file dir can't be empty")
	}

	return nil
}

//Initialize main logger
func InitGlobalLogger(writer io.Writer) error {
	dateTimeWriter := DateTimeWriterProxy{
		writer: writer,
	}
	log.SetOutput(dateTimeWriter)
	log.SetFlags(0)

	if ConfigWarn != "" {
		Warn(ConfigWarn)
	}
	return nil
}

func SystemErrorf(format string, v ...interface{}) {
	SystemError(fmt.Sprintf(format, v...))
}

func SystemError(v ...interface{}) {
	msg := []interface{}{"System error:"}
	msg = append(msg, v...)
	Error(msg...)
	notifications.SystemError(msg...)
}

func Errorf(format string, v ...interface{}) {
	Error(fmt.Sprintf(format, v...))
}

func Error(v ...interface{}) {
	log.Println(errMsg(v...))
}

func Infof(format string, v ...interface{}) {
	Info(fmt.Sprintf(format, v...))
}

func Info(v ...interface{}) {
	log.Println(append([]interface{}{infoPrefix}, v...)...)
}

func Debugf(format string, v ...interface{}) {
	Debug(fmt.Sprintf(format, v...))
}

func Debug(v ...interface{}) {
	log.Println(append([]interface{}{debugPrefix}, v...)...)
}

func Warnf(format string, v ...interface{}) {
	Warn(fmt.Sprintf(format, v...))
}

func Warn(v ...interface{}) {
	log.Println(append([]interface{}{warnPrefix}, v...)...)
}

func Fatal(v ...interface{}) {
	log.Fatal(errMsg(v...))
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, errMsg(v...))
}

func errMsg(values ...interface{}) string {
	valuesStr := []string{errPrefix}
	for _, v := range values {
		valuesStr = append(valuesStr, fmt.Sprint(v))
	}
	return color.Red.Sprint(strings.Join(valuesStr, " "))
}
