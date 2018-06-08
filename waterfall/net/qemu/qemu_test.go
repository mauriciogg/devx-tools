package qemu

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/waterfall/test_utils"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

var (
	runfiles string

	// test args
	launcher = flag.String("launcher", "", "The path to the emulator launcher")
	adbTurbo = flag.String("adb_turbo", "", "The path to abd.turbo binary")
	server   = flag.String("server", "", "The path to test server")
)

func runfilesRoot(path string) string {
	sep := "qemu_test.runfiles/__main__"
	return path[0 : strings.LastIndex(path, sep)+len(sep)]
}

func init() {
	flag.Parse()

	if *launcher == "" || *adbTurbo == "" || *server == "" {
		log.Fatalf("launcher, adb and server args need to be provided")
	}

	wd, err := os.Getwd()
	if err != nil {
		panic("unable to get wd")
	}
	runfiles = runfilesRoot(wd)
}

func testBytes(size int) []byte {
	bb := new(bytes.Buffer)
	var i uint32
	for i = 0; i < uint32(size); i += 4 {
		binary.Write(bb, binary.LittleEndian, i)
	}
	return bb.Bytes()
}

func openSocket(emuDir string) (net.Listener, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(filepath.Join(emuDir, "images", "session")); err != nil {
		return nil, err
	}
	os.Remove(socketName)

	lis, err := net.Listen("unix", socketName)
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(wd); err != nil {
		return nil, err
	}
	return lis, nil
}

func runServer(ctx context.Context, adbTurbo, adbPort, server string, n, bs int) (chan string, chan error, error) {
	s := "localhost:" + adbPort
	_, err := test_utils.ExecOnDevice(
		ctx, adbTurbo, s, "push", []string{server, "/data/local/tmp/server"})
	if err != nil {
		return nil, nil, err
	}

	_, err = test_utils.ExecOnDevice(
		ctx, adbTurbo, s, "shell", []string{"chmod", "+x", "/data/local/tmp/server"})
	if err != nil {
		return nil, nil, err
	}

	outChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		out, err := test_utils.ExecOnDevice(
			ctx, adbTurbo, s, "shell", []string{"/data/local/tmp/server",
				"-conns", strconv.Itoa(n), "-rec_n", strconv.Itoa(bs)})
		outChan <- out
		errChan <- err
	}()
	return outChan, errChan, nil
}

func setupEmu(launcher, adbServerPort, adbPort, emuPort string) (string, error) {
	emuDir, err := ioutil.TempDir("", "emulator")
	if err != nil {
		return "", err
	}

	if err := test_utils.StartEmulator(
		emuDir, launcher, adbServerPort, adbPort, emuPort); err != nil {
		return "", err
	}
	return emuDir, nil
}

func read(c net.Conn, buff []byte) ([]byte, error) {
	b := bytes.NewBuffer(buff)
	if _, err := io.Copy(bytes.NewBuffer(buff), c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func getAdbPorts() (string, string, string, error) {
	p, err := test_utils.PickUnusedPort()
	if err != nil {
		return "", "", "", err
	}
	adbServerPort := strconv.Itoa(p)

	p, err = test_utils.PickUnusedPort()
	if err != nil {
		return "", "", "", err
	}
	adbPort := strconv.Itoa(p)

	p, err = test_utils.PickUnusedPort()
	if err != nil {
		return "", "", "", err
	}
	emuPort := strconv.Itoa(p)

	return adbServerPort, adbPort, emuPort, nil
}

func TestSingleConn(t *testing.T) {
	adbServerPort, adbPort, emuPort, err := getAdbPorts()
	if err != nil {
		t.Fatal(err)
	}

	l := filepath.Join(runfiles, *launcher)
	a := filepath.Join(runfiles, *adbTurbo)
	svr := filepath.Join(runfiles, *server)

	emuDir, err := setupEmu(l, adbServerPort, adbPort, emuPort)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(emuDir)
	defer test_utils.KillEmu(l, adbServerPort, adbPort, emuPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := 4 * 1024 * 1024
	tb := testBytes(s)
	outChan, errChan, err := runServer(ctx, a, adbPort, svr, 1, s)
	if err != nil {
		t.Fatalf("error starting server: %v", err)
	}

	lis, err := openSocket(emuDir)
	if err != nil {
		t.Fatalf("error opening socket: %v", err)
	}
	defer lis.Close()

	c, err := MakeConn(lis)
	if err != nil {
		t.Fatal(err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer c.Close()
		wr, err := io.Copy(c, bytes.NewBuffer(tb))
		if err != nil {
			return err
		}
		if wr != int64(len(tb)) {
			return fmt.Errorf("wrote %d bytes but had %d", wr, len(tb))
		}
		return nil
	})

	eg.Go(func() error {
		bb := new(bytes.Buffer)
		rr, err := io.Copy(bb, c)
		if err != nil {
			return err
		}
		if rr != int64(len(tb)) {
			return fmt.Errorf("read %d but only sent %d", rr, len(tb))
		}
		if bytes.Compare(tb, bb.Bytes()) != 0 {
			return fmt.Errorf("sent bytes not the same as received")
		}
		return err
	})

	if err := eg.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	out := <-outChan
	if err := <-errChan; err != nil {
		t.Errorf("server died with error: %v out %v", err, out)
	}

}

func TestMultipleConn(t *testing.T) {
	adbServerPort, adbPort, emuPort, err := getAdbPorts()
	if err != nil {
		t.Fatal(err)
	}

	l := filepath.Join(runfiles, *launcher)
	a := filepath.Join(runfiles, *adbTurbo)
	svr := filepath.Join(runfiles, *server)

	emuDir, err := setupEmu(l, adbServerPort, adbPort, emuPort)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(emuDir)
	defer test_utils.KillEmu(l, adbServerPort, adbPort, emuPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conns := 256
	s := 4 * 1024 * 1024
	tb := testBytes(s)
	outChan, errChan, err := runServer(ctx, a, adbPort, svr, conns, s)
	if err != nil {
		t.Fatalf("error starting server: %v", err)
	}

	lis, err := openSocket(emuDir)
	if err != nil {
		t.Fatalf("error opening socket: %v", err)
	}
	defer lis.Close()

	eg, _ := errgroup.WithContext(ctx)
	for i := 0; i < conns; i++ {

		// Lets avoid variable aliasing
		func() {
			c, err := MakeConn(lis)
			if err != nil {
				t.Fatal(err)
			}

			eg.Go(func() error {
				defer c.Close()
				wr, err := io.Copy(c, bytes.NewBuffer(tb))
				if err != nil {
					return err
				}
				if wr != int64(len(tb)) {
					return fmt.Errorf("wrote %d bytes but had %d", wr, len(tb))
				}
				return nil
			})

			eg.Go(func() error {
				bb := new(bytes.Buffer)
				rr, err := io.Copy(bb, c)
				if err != nil {
					return err
				}
				if rr != int64(len(tb)) {
					return fmt.Errorf("read %d but only sent %d", rr, len(tb))
				}
				if bytes.Compare(tb, bb.Bytes()) != 0 {
					return fmt.Errorf("sent bytes not the same as received")
				}
				return err
			})
		}()

	}

	if err := eg.Wait(); err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	out := <-outChan
	if err := <-errChan; err != nil {
		t.Errorf("got unexpected error: %v out %v", err, out)
	}
}
