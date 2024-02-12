package server

import (
	"flag"
	"fmt"
	"net"
	"os"
)

type UDSOptions struct {
	ListenUds  bool
	SocketPath string
}

func InitUDSOptions(fs *flag.FlagSet, usageStr *string) *UDSOptions {
	opts := &UDSOptions{}

	fs.StringVar(&opts.SocketPath, "uds", "", "Listen for connections domain socket.")
	fs.StringVar(&opts.SocketPath, "udsfile", "", "Listen for connections domain socket.")

	*usageStr += `
Domain Socket Options:
    -uds, --udsfile <file>           Path to unix domain socket 
`
	return opts
}

func ConfigureUDSOptions(opts *UDSOptions) error {
	fmt.Printf("Socket path: %s", opts.SocketPath)
	if opts.SocketPath != _EMPTY_ {
		opts.ListenUds = true
	}

	return nil
}

func (s *Server) StartUds(socketPath string) {
	if s.isShuttingDown() {
		return
	}

	sOpts := s.getOpts()
	if sOpts.DontListen {
		return
	}

	// ...if it's decided to support UDS, remove socket path from parameter list.
	/*
		// Maybe
		o := &sopts.Uds
		var socketPath := o.SocketPath
	*/

	info, err := os.Stat(socketPath)
	if err != nil {
		// if it's not an "not exist" err, we don't care
		if !os.IsNotExist(err) {
			s.Fatalf("Cannot open UDS, %v", err)
			return
		}
	} else if info.IsDir() {
		// we won't try to delete directories.
		s.Fatalf("Cannot open UDS, '%s' already exists as a directory.", socketPath)
	}

	if info != nil {
		err = os.Remove(socketPath)
		if err != nil {
			s.Fatalf("Cannot open UDS, %v", err)
			return
		}
	}

	s.mu.Lock()
	hl, err := natsListen("unix", socketPath)

	// TODO: how do we get notified of server shutdown, so that we can delete the file.

	if err != nil {
		s.mu.Unlock()
		s.Fatalf("Unable to listen for UDS connections: %v", err)
		return
	}

	go s.acceptConnections(hl, "UDS", func(conn net.Conn) { s.createClientInProcess(conn) }, nil)
	s.Noticef("Listening for clients on %s", hl.Addr().String())

	s.mu.Unlock()

	go func() {
		<-s.quitCh
		s.Tracef("Removing UDS socket: %s", hl.Addr().String())

		err = os.Remove(socketPath)
		err = os.Remove(socketPath)
		if err != nil {
			s.Tracef("Failed to remove UDS, %v", err)
			return
		}
	}()
}
