package transport

import (
	"fmt"
	"os"
)

// SerialTransport writes ESC/POS bytes straight to a serial device. Most 58mm
// "USB" thermal printers expose a USB-serial interface that appears as
// /dev/ttyUSB0, /dev/ttyACM0 (Linux) or COM3 (Windows); writing bytes to that
// device drives the printer. This avoids native USB (libusb) dependencies.
//
// True USB (vendor/product discovery) is a future enhancement; see the design doc.
type SerialTransport struct {
	device string
}

// NewSerial builds a serial transport for the given device path.
func NewSerial(device string) (*SerialTransport, error) {
	if device == "" {
		return nil, fmt.Errorf("serial transport requires PRINT_SERIAL_DEVICE (e.g. /dev/ttyUSB0)")
	}
	return &SerialTransport{device: device}, nil
}

// Write sends the bytes to the serial device.
func (s *SerialTransport) Write(jobID string, data []byte) error {
	f, err := os.OpenFile(s.device, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// Close is a no-op for the serial transport.
func (s *SerialTransport) Close() error { return nil }

// Type reports the transport kind.
func (s *SerialTransport) Type() string { return "serial" }

// Health reports whether the serial device node exists.
func (s *SerialTransport) Health() error {
	if _, err := os.Stat(s.device); err != nil {
		return err
	}
	return nil
}
