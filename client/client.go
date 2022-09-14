package client

import (
	"bufio"
	pbf "chatroom/protobuf"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

type connectData struct {
	protocol string
	ip       string
	prot     string
}

type msgType struct {
	nick string
	send string
	quit string
}

var m_connectData connectData
var m_usreNick string
var m_sel string

func init() {
	m_connectData = connectData{
		protocol: "tcp",
		ip:       "127.0.0.1",
		prot:     "4444",
	}
	m_usreNick = ""
	m_sel = "0"
}

type clientData struct {
	userName, passWord string
}

func main() {
	conn, err := net.Dial(m_connectData.protocol, m_connectData.ip+":"+m_connectData.prot)
	if err != nil {
		fmt.Println("连接失败: ", err)
		return
	}
	defer conn.Close()

	loginState := false
	for !loginState {
		fmt.Println("请选择：1.登录 2.注册 3.退出")
		fmt.Scanf("%s", &m_sel)
		switch m_sel {
		case "1":
			loginState = login(conn)
		case "2":
			if register(conn) {
				loginState = login(conn)
			}
		case "3":
			return
		}
	}

	fmt.Println("登陆成功进聊天室...")
	datas, err := serializeMsg(m_usreNick, "", pbf.MsgC2S_join)
	sendMsgData := msgData{protoData: datas, types: 2}
	conn.Write(sendMsgData.encode())
	go recvHandle(conn)
	sendHandle(conn)
}

// state : 1 登录 2注册 // types : 消息类 1 注册类型 2 消息类型 3 注册响应类
func setAccountData(userName, passWord string, state int32, types uint32) []byte {

	accountData := &pbf.Account{
		User:  userName,
		Pass:  passWord,
		State: state,
	}

	bf, _ := proto.Marshal(accountData)
	data := msgData{protoData: bf, types: types}

	return data.encode()
}

// 登录逻辑
func login(conn net.Conn) bool {
	var userName string
	var passWord string
	for {
		fmt.Println("请输入用户名:")
		fmt.Scanf("%s", &userName)
		fmt.Println("请输入密码:")
		fmt.Scanf("%s", &passWord)

		data := setAccountData(userName, passWord, 1, uint32(1))
		conn.Write(data) // 同步阻塞
		msgDatas := recv(conn)
		if msgDatas.types == 3 {
			recvDatas := &pbf.AccountAck{}
			proto.Unmarshal(msgDatas.protoData, recvDatas)
			if recvDatas.Ack == 100 { // 登录成功
				m_usreNick = userName
				return true
			} else if recvDatas.Ack == 200 {
				fmt.Println("账号或密码错误")
			}
		}
		return false
	}
}

// 注册逻辑
func register(conn net.Conn) bool {

	var userName string
	var passWord string
	var passWord2 string

	for {
		fmt.Println("请输入注册用户名:")
		fmt.Scanf("%s", &userName)
		fmt.Println("请输入密码:")
		fmt.Scanf("%s", &passWord)
		fmt.Println("请输入再次确认密码:")
		fmt.Scanf("%s", &passWord2)
		if passWord != passWord2 {
			fmt.Println("两次密码不一致 请重试")
			continue
		}
		data := setAccountData(userName, passWord, 2, uint32(1))
		conn.Write(data)
		msgDatas := recv(conn)
		if msgDatas.types == 3 {
			recvDatas := &pbf.AccountAck{}
			proto.Unmarshal(msgDatas.protoData, recvDatas)
			if recvDatas.Ack == 100 { // 成功
				fmt.Println("注册成功")
				return true
			} else if recvDatas.Ack == 300 {
				fmt.Println("已被注册")
			}
		}
		return false
	}
}

// 序列化消息
func serializeMsg(nick, msg string, msgType pbf.MsgC2S_MsgType) ([]byte, error) {

	res := &pbf.MsgC2S{
		Nick:    m_usreNick,
		Msg:     msg,
		Time:    time.Now().Unix(),
		MsgType: msgType,
	}
	datas, err := proto.Marshal(res)

	return datas, err

}

// 发送消息处理
func sendHandle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" || err == io.EOF {
			send(pbf.MsgC2S_quit, conn, input)
			return
		}

		send(pbf.MsgC2S_send, conn, input)
	}
}

func send(msgType pbf.MsgC2S_MsgType, conn net.Conn, content string) {
	datas, err := serializeMsg(m_usreNick, content, msgType)
	if err != nil {
		fmt.Println("序列化错误")
	}
	data := msgData{protoData: datas, types: 2}
	conn.Write(data.encode())
}

// 接收消息处理
func recvHandle(conn net.Conn) {
	for {
		recvDatas := recv(conn)
		if recvDatas.types == 2 {
			msg := &pbf.MsgC2S{}
			err := proto.Unmarshal(recvDatas.protoData, msg)
			if err != nil {
				fmt.Println("client : 解析错误")
				continue
			}
			printHandle(msg, conn)
		}

	}
}

func recv(conn net.Conn) msgData {
	buf := [256]byte{}
	len, err := conn.Read(buf[:])
	if err != nil || len == 0 {
		fmt.Println("read err:", err)
	}

	recvBuf := buf[:len]
	msgdatas := msgData{protoData: recvBuf}
	recvDatas := msgdatas.decode()
	return recvDatas
}

// 界面打印处理
func printHandle(msg *pbf.MsgC2S, conn net.Conn) {

	nick := msg.GetNick()

	var res string

	switch msg.MsgType {

	case pbf.MsgC2S_join:
		res = "[system] : " + nick + " 加入聊天室"

	case pbf.MsgC2S_send:
		res = "[" + Unix2string(msg.GetTime()) + "] " + nick + ":" + msg.GetMsg()

	case pbf.MsgC2S_quit:
		res = "[system] : " + nick + " 退出聊天室"

	}

	fmt.Println(res)

}

func Unix2string(unix int64) string {
	return time.Unix(unix, 0).Format("2006-01-02 15:04:05")
}

// types : 消息类 1 注册类型 2 消息类型 3 注册响应类
type msgData struct {
	size      uint32
	types     uint32
	protoData []byte
}

func (this *msgData) encode() []byte {
	this.size = uint32(8 + len(this.protoData))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], this.size)
	binary.BigEndian.PutUint32(buf[4:8], this.types)
	buf = append(buf, this.protoData...)
	protoBuf, _ := proto.Marshal(&pbf.MsgData{Datas: buf})
	return protoBuf
}

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
