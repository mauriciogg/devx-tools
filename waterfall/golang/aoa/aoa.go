// Packages aoa has utilities to configure and talk to an Android device as AoA (Android open Accessory).
package aoa

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gousb"
)

const (
	// Taken from device/google/marlin/init.common.usb.rc
	// Google vendor id when in accessory mode
	googleVendorID = 0x18D1

	manufacturer = "devx"
	model        = "waterfall"
	description  = "Waterfall USB"
	version      = "1.0"
	serial       = "00000012345678"
	uri          = "http://play.google.com/store/apps/details?id=com.google.waterfall.usb"

	aoaClass    = 255
	aoaSubclass = 255
	aoaProtocol = 0

)

var (
	// Product ID to use when in accessory mode + ADB
	usbAccessoryADBProductID int = 0x2D01

	// Product ids with accessory mode enabled.
	usbAccessoryIds = map[int]bool{
		0x2D00: true,
		0x2D01: true,
		0x2D04: true,
		0x2D05: true,
	}

	// vendor specific requests
	vendorInXfer  uint8 = 0xC0
	vendorOutXfer uint8 = 0x40

	// requests
	getProtocolRequest    uint8 = 51
	sendStringRequest     uint8 = 52
	startAccessoryRequest uint8 = 53

	manufacturerIDX uint16 = 0
	modelIDX        uint16 = 1
	descriptionIDX  uint16 = 2
	versionIDX      uint16 = 3
	uriIDX          uint16 = 4
	serialIDX       uint16 = 5

)

type ReadWriter struct {
	intf *gousb.Interface
	in   *gousb.InEndpoint
	out  *gousb.OutEndpoint
}

func (u *ReadWriter) Read(in []byte) (int, error) {
	return u.in.Read(in)
}

func (u *ReadWriter) Write(out []byte) (int, error) {
	return u.out.Write(out)
}

func sendString(dev *gousb.Device, idx uint16, str string) error {
	_, err := dev.Control(vendorOutXfer, sendStringRequest, 0, idx, []byte(str))
	return err
}

// switchToAccessory switched the USB device to Accessory mode following the protocol
// described in https://source.android.com/devices/accessories/aoa2.
func switchToAccessory(dev *gousb.Device) error {
	if err := sendString(dev, manufacturerIDX, manufacturer); err != nil {
		return err
	}
	if err := sendString(dev, modelIDX, model); err != nil {
		return err
	}
	if err := sendString(dev, descriptionIDX, description); err != nil {
		return err
	}
	if err := sendString(dev, versionIDX, version); err != nil {
		return err
	}
	if err := sendString(dev, uriIDX, uri); err != nil {
		return err
	}
	if err := sendString(dev, serialIDX, serial); err != nil {
		return err
	}
	if _, err := dev.Control(vendorOutXfer, startAccessoryRequest, 0, 0, nil); err != nil {
		return err
	}
	return nil
}


func makeReadWriter(i *gousb.Interface) (*ReadWriter, error) {
	rw := &ReadWriter{intf: i}
	for _, desc := range i.Setting.Endpoints {
		switch desc.Direction {
		case gousb.EndpointDirectionIn:
			in, err := i.InEndpoint(desc.Number)
			if err != nil {
				return nil, err
			}
			rw.in = in
		case gousb.EndpointDirectionOut:
			out, err := i.OutEndpoint(desc.Number)
			if err != nil {
				return nil, err
			}
			rw.out = out
		}
	}
	return rw, nil
}


// getAccessoryInterface traverses the device tree and returns the AOA interface.
func getAccessoryInterface(dev *gousb.Device) (*gousb.Interface, error) {
	c, err := dev.ActiveConfigNum()
	if err != nil {
		return nil, err
	}

	config, err := dev.Config(c)
	if err != nil {
		return nil, err
	}

	// find the relevant interface (accessory) -> config.Interface(i)
	var aoaIS gousb.InterfaceSetting
	found := false
	for _, id := range dev.Desc.Configs[c].Interfaces {
		for _, is := range id.AltSettings {
			if is.Class == aoaClass && is.SubClass == aoaSubclass && is.Protocol == aoaProtocol {
				aoaIS = is
				found = true
				break
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("couldn't find AOA interface")
	}

	return config.Interface(aoaIS.Number, aoaIS.Alternate)

}

// findDevice traverses the USB an searches for the device with serial.
// On success it returns a handle the the USB device. The caller is responsible for disposing of the device.
// If the the device is not found it returns an error.
func FindDevice(serial string) (*gousb.Device, error) {
	ctx := gousb.NewContext()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == googleVendorID
	})

	if err != nil {
		return nil, err
	}

	// remeber to close devices we are not interested in
	defer func() {
		for _, dev := range devs {
			if dev == nil {
				continue
			}
			if err := dev.Close(); err != nil {
				// Just log. Nothing else we can do.
				log.Printf("Error closing device: %v", err)
			}
		}
	}()

	var d *gousb.Device
	for i, dev := range devs {
		s, err := dev.SerialNumber()
		if err != nil {
			return nil, err
		}

		if serial == s {
			d = dev
			devs[i] = nil
			break
		}
	}

	if d == nil {
		return nil, fmt.Errorf("device with serial %s not found", serial)
	}
	return d, nil
}

func ConfigureAOA(serial string) (dev *gousb.Device, err error) {
	dev, err = FindDevice(serial)
	if err != nil {
		return nil, err
	}

	if !usbAccessoryIds[int(dev.Desc.Product)] {
		if err = switchToAccessory(dev); err != nil {
			dev.Close()
			return nil, err
		}
	}
	// Close and query for the device to get see latest config.
	if err = dev.Close(); err != nil {
		return nil, err
	}

	// Try a few times before giving up.
	for i := 0; i<3; i++ {
		log.Println("Attempting to connect to device.")
		time.Sleep(time.Millisecond * 500)
		dev, err = FindDevice(serial)
		if err == nil {
			break
		}
	}
	return dev, err
}

func Connect(serial string) (*ReadWriter, error) {
	dev, err := ConfigureAOA(serial)
	if err != nil {
		return nil, err
	}

	intf, err := getAccessoryInterface(dev)
	if err != nil {
		dev.Close()
		return nil, err
	}
	return makeReadWriter(intf)
}
