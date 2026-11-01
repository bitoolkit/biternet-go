package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/bitoolkit/biternet-go/internal/desktop"
)

func main() {
	silent := flag.Bool("silent", false, "启动到系统托盘")
	version := flag.Bool("version", false, "显示版本")
	// MyGo adds this argument to the OS login item.
	flag.Bool("mygo-opened-at-login", false, "由系统登录项启动")
	flag.Parse()

	if *version {
		fmt.Println("BITERNET v" + desktop.Version())
		return
	}

	if err := desktop.Run(*silent); err != nil {
		log.Fatal(err)
	}
}
