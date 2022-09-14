package Server

import (
	"bufio"
	db "chatroom/mongodb"
	pbf "chatroom/protobuf"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/proto"
)

var m_userList map[string]net.Conn
var m_connectData connectData
var client *mongo.Client

type UserData struct {
	Name         string
	PassWord     string
	RegistryTime int64
}

type connectData struct {
	protocol string
	ip       string
	prot     string
}

func init() {

	m_connectData = connectData{
		protocol: "tcp",
		ip:       "127.0.0.1",
		prot:     "4444",
	}
	m_userList = make(map[string]net.Conn)
}

// types : 消息类 1 注册类型 2 消息类型 3 注册响应类
type msgData struct {
	size      uint32
	types     uint32
	protoData []byte
}

// 编码
func (this *msgData) encode() []byte {
	this.size = uint32(8 + len(this.protoData))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], this.size)
	binary.BigEndian.PutUint32(buf[4:8], this.types)
	buf = append(buf, this.protoData...)
	protoBuf, _ := proto.Marshal(&pbf.MsgData{Datas: buf})
	return protoBuf
}

// 解码
func (this *msgData) decode() msgData {
	recvMsgData := &pbf.MsgData{}
	proto.Unmarshal(this.protoData, recvMsgData)

	recvBuf := recvMsgData.GetDatas()
	this.size = binary.BigEndian.Uint32(recvBuf[0:4])
	// fmt.Println(int(this.size) == len(buf))
	this.types = binary.BigEndian.Uint32(recvBuf[4:8])
	this.protoData = recvBuf[8:]

	return *this
}

func Start() {

	listener, err := net.Listen(m_connectData.protocol, m_connectData.ip+":"+m_connectData.prot)
	if err != nil {
		fmt.Println("connect err!!!", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept one of err:", err)
			continue
		}
		println("connect a new client ...")
		go clientHandle(conn)
	}
}

// 客户端响应处理
func clientHandle(conn net.Conn) {
	defer conn.Close()
	for {
		var buf [256]byte
		reader := bufio.NewReader(conn)
		len, err := reader.Read(buf[:])
		if err != nil || len == 0 {
			continue
		}
		recvBuf := buf[:len]
		msgdatas := msgData{protoData: recvBuf}
		recvDatas := msgdatas.decode()

		if recvDatas.types == 2 { // 消息处理
			msgTypeHandel(conn, recvDatas.protoData, recvBuf) //recvBuf原始数据作为消息广播
		}

		if recvDatas.types == 1 { // 注册登录处理
			registerHandel(conn, recvDatas.protoData)
		}

	}
}

// 注册登录处理
func registerHandel(conn net.Conn, protoData []byte) {
	account := &pbf.Account{}
	proto.Unmarshal(protoData, account)
	ackCode := 0
	switch account.GetState() {
	// 登录
	case 1:
		if db.Login(account.GetUser(), account.GetPass()) {
			ackCode = 100
		} else {
			ackCode = 200
		}
	// 注册
	case 2:
		if !db.CheckUserNameExist(account.GetUser()) {
			if db.Register(UserData{
				Name:         account.GetUser(),
				PassWord:     account.GetPass(),
				RegistryTime: time.Now().Unix(),
			}) {
				ackCode = 100
			} else {
				ackCode = 200
			}
		} else {
			ackCode = 300
		}
	}

	AccountAck := &pbf.AccountAck{Ack: int32(ackCode)} //100：成功 200：失败
	accountBuf, _ := proto.Marshal(AccountAck)
	accountData := msgData{protoData: accountBuf, types: 3} // 登录注册响应
	conn.Write(accountData.encode())

}

// 消息处理
func msgTypeHandel(conn net.Conn, protoData, datas []byte) {
	msg := &pbf.MsgC2S{}
	err := proto.Unmarshal(protoData, msg)
	if err != nil {
		fmt.Println("解析错误")
		return
	}
	nick := msg.GetNick()
	switch msg.MsgType {
	case pbf.MsgC2S_join:
		broadcast(nick, datas, func() {
			m_userList[nick] = conn
		})
	case pbf.MsgC2S_send:
		broadcast(nick, datas, nil)

	case pbf.MsgC2S_quit:
		broadcast(nick, datas, func() {
			delete(m_userList, nick)
		})
	}
}

// 广播
func broadcast(nick string, datas []byte, callback func()) {
	for k, v := range m_userList {
		if k != nick {
			v.Write(datas)
		}
	}
	if callback != nil {
		callback()
	}
}
