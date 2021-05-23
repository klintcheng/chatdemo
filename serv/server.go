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
	loginwait time.Duration //登陆超时
	readwait  time.Duration //读超时
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
			loginwait: time.Second * 10,
			readwait:  time.Minute * 3,
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
			resp(w, http.StatusBadRequest, err.Error())
			return
		}
		// 读取userId
		user := r.URL.Query().Get("user")
		if user == "" {
			resp(w, http.StatusBadRequest, "user is required")
			return
		}
		// 添加到会话管理中
		old, ok := s.addUser(user, conn)
		if ok {
			// 断开旧的连接
			old.Close()
		}
		log.Infof("user %s in", user)

		go func(user string, conn net.Conn) {
			err := s.readloop(user, conn)
			if err != nil {
				log.Error(err)
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
	if ok {
		return old, true
	}
	s.users[user] = conn
	return nil, false
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

// string connID
// []byte data
func (s *Server) WriteText(conn net.Conn, data []byte) error {
	// 创建文本帧数据
	f := ws.NewTextFrame(data)
	return ws.WriteFrame(conn, f)
}

func resp(w http.ResponseWriter, code int, body string) {
	w.WriteHeader(code)
	if body != "" {
		_, _ = w.Write([]byte(body))
	}
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
func (s *Server) handle(user string, recv string) {
	s.RLock()
	defer s.RUnlock()
	broadcast := fmt.Sprintf("%s -- FROM %s", recv, user)
	for u, conn := range s.users {
		if u == user { // 不发给自己
			continue
		}
		err := s.WriteText(conn, []byte(broadcast))
		if err != nil {
			logrus.Errorf("write to %s failed, error: %v", user, err)
		}
	}
}
