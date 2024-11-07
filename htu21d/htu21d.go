package htu21d

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"

	"periph.io/x/conn/v3/physic"
)

type Opts struct {
	HoldMaster bool
	Config     uint8 // User register
}

const (
	HoldMaster          = 0xE5
	NoHoldMaster        = 0xF5
	SenseBits           = byte(0b00000011)
	ReadDelay           = time.Millisecond * 100
	StatusOKTemperature = 1
	StatusOKHumidity    = 2
)

func NewI2C(b i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}}
	if err := d.makeDev(opts); err != nil {
		return nil, err
	}
	return d, nil
}

type Dev struct {
	d    conn.Conn
	opts Opts
	Name string
	mu   sync.Mutex
}

func (d *Dev) makeDev(opts *Opts) error {
	d.opts = *opts
	return nil
}

func (d *Dev) SenseHumidity(e *physic.Env) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	read := make([]byte, 3)
	// send command, read three byte response
	var err error

	if d.opts.HoldMaster {
		// Hold master
		command := []byte{HoldMaster}
		err = d.d.Tx(command, nil)
		if err != nil {
			return errors.Join(errors.New("Failed to send command"), err)
		}
		err = d.d.Tx(nil, read)
		if err != nil {
			return errors.Join(errors.New("Failed to read response"), err)
		}
	} else {
		// No hold master
		command := []byte{NoHoldMaster}
		err = d.d.Tx(command, nil)
		if err != nil {
			return errors.Join(errors.New("Failed to send command"), err)
		}
		// Give time for device to complete measurement
		time.Sleep(time.Millisecond * 100)
		err = d.d.Tx(nil, read)
		if err != nil {
			return errors.Join(errors.New("Failed to read response"), err)
		}
	}

	msb := read[0]
	lsb := read[1]

	// read status bits
	status := lsb & SenseBits

	if status != StatusOKHumidity {
		return fmt.Errorf("Status was not correct [%v]", status)
	}

	// set status bits to zero - see spec
	lsb = lsb & ^SenseBits
	// Raw value is two bytes
	hRaw := int32(msb)<<8 | int32(lsb)
	rhTemp := float64(hRaw) * (125.0 / 65536.0) // magic numbers - see spec
	rh := float64(rhTemp - 6.0)                 // magic numbers - see spec

	e.Humidity = physic.RelativeHumidity(rh * float64(physic.PercentRH))

	return err
}

func (d *Dev) SenseTemperature(e *physic.Env) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return errors.New("Not implemented")
}

func (d *Dev) SetOptions() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return errors.New("Not implemented")
}
