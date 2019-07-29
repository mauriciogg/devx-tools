// aoa_utils_bin is an utility program to perform aoa opearations on a device.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/waterfall/golang/aoa"
)

var (
	serial = flag.String("serial", "", "Serial number of USB device to print information.")
	cmd = flag.String("cmd", "info", "info displays info about the usb device.")
)

func info(serial string) error {
	d, err := aoa.FindDevice(serial)
	if err != nil {
		return err
	}
	defer d.Close()

	log.Printf("Found device: %s\n", d.String())
	return nil
}

func switchConfig(serial string) error {
	dev, err := aoa.ConfigureAOA(serial)
	if err != nil {
		return err
	}
	return dev.Close()
}

func reset(serial string) error {
	d, err := aoa.FindDevice(serial)
	if err != nil {
		return err
	}
	return d.Reset()
}

func echo(serial string) error {
	rw, err := aoa.Connect(serial)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 256)
	for s.Scan() {
		t := s.Text()

		if t == "quit" || t == "exit" || t == "" {
			break
		}

		_, err := rw.Write([]byte(t))
		if err != nil {
			return err
		}
		fmt.Printf(">> ")
		n, err := rw.Read(buf)
		if err != nil {
			return err
		}
		fmt.Printf(">>> %s\n", string(buf[0:n]))
	}
	return nil
}

func main() {
	flag.Parse()

	if *serial == "" {
		log.Fatalf("Need to specify -serial.")
	}

	var err error
	switch *cmd {
	case "info":
		err = info(*serial)
	case "switch":
		err = switchConfig(*serial)
	case "reset":
		err = reset(*serial)
	case "echo":
		err = echo(*serial)
	default:
		log.Fatalf("%s is not a valid command.", *cmd)
	}

	if err != nil {
		log.Fatalf("Error running cmd: %v\n", err)
	}
}
