package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// StartOptions StartOptions
type StartOptions struct {
	address string
	user    string
}

// NewCmd NewCmd
func NewCmd(ctx context.Context) *cobra.Command {
	opts := &StartOptions{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Start client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(ctx, opts)
		},
	}
	cmd.PersistentFlags().StringVarP(&opts.address, "address", "a", "ws://127.0.0.1:8000", "server address")
	cmd.PersistentFlags().StringVarP(&opts.user, "user", "u", "", "user")
	return cmd
}

func run(ctx context.Context, opts *StartOptions) error {
	url := fmt.Sprintf("%s?user=%s", opts.address, opts.user)
	logrus.Info("connect to ", url)

	han, err := connect(url)
	if err != nil {
		return err
	}
	go func() {
		for msg := range han.recv {
			logrus.Info("Receive message:", string(msg))
		}
	}()

	tk := time.NewTicker(time.Second * 6)
	for {
		select {
		case <-tk.C:
			err := han.sendText("hello")
			if err != nil {
				logrus.Error(err)
			}
		case <-han.close:
			logrus.Printf("connection closed")
			return nil
		}
	}
}

type handler struct {
	conn  net.Conn
	close chan struct{}
	recv  chan []byte
}

// Disconnect Disconnect
func (h *handler) Disconnect() chan struct{} {
	return h.close
}

func (h *handler) Receive(data []byte) chan []byte {
	return h.recv
}

func connect(addr string) (*handler, error) {
	_, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	conn, _, _, err := ws.Dial(context.Background(), addr)
	if err != nil {
		return nil, err
	}

	h := handler{
		conn:  conn,
		close: make(chan struct{}, 1),
		recv:  make(chan []byte, 10),
	}

	go func() {
		err := h.readloop(conn)
		if err != nil {
			logrus.Warn(err)
		}
		// 通知上层
		h.close <- struct{}{}
	}()

	return &h, nil
}

func (h *handler) readloop(conn net.Conn) error {
	logrus.Info("readloop started")
	for {
		frame, err := ws.ReadFrame(conn)
		if err != nil {
			return err
		}
		if frame.Header.OpCode == ws.OpPong {
			continue
		}
		if frame.Header.OpCode == ws.OpClose {
			return errors.New("remote side close the channel")
		}
		if frame.Header.OpCode == ws.OpText {
			h.recv <- frame.Payload
		}
	}
}

func (h *handler) sendText(msg string) error {
	logrus.Info("send message :", msg)
	return wsutil.WriteClientText(h.conn, []byte(msg))
}
