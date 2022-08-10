package config

import (
	"fmt"

	"github.com/spf13/viper"
)

var LzqConfig = &viper.Viper{}

func LzqConfigInit() {
	LzqConfig = viper.New()
	LzqConfig.AddConfigPath("./config/") // 文件所在目录
	LzqConfig.SetConfigName("config")    // 文件名
	LzqConfig.SetConfigType("ini")       // 文件类型

	if err := LzqConfig.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("找不到配置文件..")
		} else {
			fmt.Println("配置文件出错..")
		}
	}
}
