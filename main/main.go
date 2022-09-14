package main

import (
	db "chatroom/mongodb"
	Server "chatroom/server"
)

func main() {
	db.LoadDb()
	Server.Start()
}
