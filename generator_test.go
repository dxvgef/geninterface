package main

import (
	"testing"
)

// Config 配置参数
type Config struct {
	Debug    bool
	Database struct {
		Addr     string
		User     string
		Password string
		Name     string
	}
	Redis Redis
}

type Redis struct {
	Addr     string
	Username string
	Password string
	DB       uint8
	Logger
}

type Logger struct {
	Level string
}

func Test_generator(t *testing.T) {
	err := Generator("generator_test.go", "Config", false)
	if err != nil {
		t.Fatal(err)
	}
}
