package serv

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/sirupsen/logrus"
)

// ServerOptions ServerOptions
type ServerOptions struct {
	writewait time.Duration //写超时时间
	readwait  time.Duration //读超时时间
}

// Server is a websocket implement of the Server
type Server struct {
	once    sync.Once
	options ServerOptions
	id      string
	address string
	sync.RWMutex
	// 会话列表
	users map[string]net.Conn
}

// NewServer NewServer
func NewServer(id, address string) *Server {
	return newServer(id, address)
}

func newServer(id, address string) *Server {
	return &Server{
		id:      id,
		address: address,
		users:   make(map[string]net.Conn, 100),
		options: ServerOptions{
			writewait: time.Second * 3,
			readwait:  time.Minute * 30,
		},
	}
}

// Start server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	log := logrus.WithFields(logrus.Fields{
		"module": "Server",
		"listen": s.address,
		"id":     s.id,
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			conn.Close()
			return
		}
		// 读取userId
		user := r.URL.Query().Get("user")
		if user == "" {
			conn.Close()
			return
		}

		// 添加到会话管理中
		old, ok := s.addUser(user, conn)
		if ok {
			// 断开旧的连接
			old.Close()
			log.Infof("close old connection %v", old.RemoteAddr())
		}
		log.Infof("user %s in from %v", user, conn.RemoteAddr())

		go func(user string, conn net.Conn) {
			err := s.readloop(user, conn)
			if err != nil {
				log.Warn(err)
			}
			conn.Close()
			// 删除用户
			s.delUser(user)

			log.Infof("connection of %s closed", user)
		}(user, conn)
	})
	log.Infoln("started")
	return http.ListenAndServe(s.address, mux)
}

func (s *Server) addUser(user string, conn net.Conn) (net.Conn, bool) {
	s.Lock()
	defer s.Unlock()
	old, ok := s.users[user]
	s.users[user] = conn
	return old, ok
}

func (s *Server) delUser(user string) {
	s.Lock()
	defer s.Unlock()
	delete(s.users, user)
}

// Shutdown Shutdown
func (s *Server) Shutdown() {
	s.once.Do(func() {
		s.Lock()
		defer s.Unlock()
		for _, conn := range s.users {
			conn.Close()
		}
	})
}

func (s *Server) readloop(user string, conn net.Conn) error {
	for {
		frame, err := ws.ReadFrame(conn)
		if err != nil {
			return err
		}
		if frame.Header.OpCode == ws.OpClose {
			return errors.New("remote side close the conn")
		}

		if frame.Header.Masked {
			ws.Cipher(frame.Payload, frame.Header.Mask, 0)
		}
		// 接收文本帧内容
		if frame.Header.OpCode == ws.OpText {
			go s.handle(user, string(frame.Payload))
		}
	}
}

// 广播消息
func (s *Server) handle(user string, message string) {
	s.RLock()
	defer s.RUnlock()
	broadcast := fmt.Sprintf("%s -- FROM %s", message, user)
	for u, conn := range s.users {
		if u == user { // 不发给自己
			continue
		}
		err := s.writeText(conn, broadcast)
		if err != nil {
			logrus.Errorf("write to %s failed, error: %v", user, err)
		}
	}
}

func (s *Server) writeText(conn net.Conn, message string) error {
	// 创建文本帧数据
	f := ws.NewTextFrame([]byte(message))
	err := conn.SetWriteDeadline(time.Now().Add(s.options.writewait))
	if err != nil {
		return err
	}
	return ws.WriteFrame(conn, f)
}
