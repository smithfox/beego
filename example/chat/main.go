package main

import (
	"github.com/smithfox/beego"
	"github.com/smithfox/beego/example/chat/controllers"
)

func main() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/ws", &controllers.WSController{})
	beego.Run()
}
