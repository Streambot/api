package main

import (
	"github.com/hoisie/web"
)

func TestHello() {
	web.Get("/", func() string {
		return "Hello world!"
	})
	web.Run(":8080")
}
