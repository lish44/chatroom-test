package main

import (
	db "chatroom/mongodb"
	Server "chatroom/server"
	"fmt"
)

func main() {
	if !db.LoadDb() {
		fmt.Println("数据库连接失败")
		return
	}
	Server.Start()
}
