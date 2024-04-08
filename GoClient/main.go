package main

import (
	"GoClient/Server"
	"Tools/Build"
)



func main() {
	Build.CheckArgs()
	Server.Serve()
}