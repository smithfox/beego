package main

import (
	"github.com/smithfox/beego"
	"github.com/smithfox/beego/example/beeapi/controllers"
)

//		Objects

//	URL					HTTP Verb				Functionality
//	/object				POST					Creating Objects
//	/object/<objectId>	GET						Retrieving Objects
//	/object/<objectId>	PUT						Updating Objects
//	/object				GET						Queries
//	/object/<objectId>	DELETE					Deleting Objects

func main() {
	beego.RESTRouter("/object", &controllers.ObjectController{})
	beego.Run()
}
