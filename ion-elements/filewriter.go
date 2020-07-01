package elements

import (
	"os"

	"github.com/sssgun/ion/pkg/log"
	"github.com/sssgun/ion/pkg/process"
	"github.com/sssgun/ion/pkg/process/samples"
)

const (
	// TypeFileWriter .
	TypeFileWriter = "FileWriter"
)

// FileWriterConfig .
type FileWriterConfig struct {
	ID   string
	Path string
}

// FileWriter instance
type FileWriter struct {
	id   string
	file *os.File
}

// NewFileWriter instance
func NewFileWriter(config FileWriterConfig) *FileWriter {
	w := &FileWriter{
		id: config.ID,
	}

	f, err := os.OpenFile(config.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		panic(err)
	}

	w.file = f

	log.Infof("NewFileWriter with config: %+v", config)

	return w
}

// Type for FileWriter
func (w *FileWriter) Type() string {
	return TypeFileWriter
}

func (w *FileWriter) Write(sample *samples.Sample) error {
	_, err := w.file.Write(sample.Payload)
	return err
}

func (w *FileWriter) Read() <-chan *samples.Sample {
	return nil
}

// Attach attach a child element
func (w *FileWriter) Attach(e process.Element) error {
	return ErrAttachNotSupported
}

// Close FileWriter
func (w *FileWriter) Close() {
	log.Infof("FileWriter.Close() %s", w.id)
}
