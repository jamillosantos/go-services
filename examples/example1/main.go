package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jamillosantos/go-services"
)

func sleep() {
	time.Sleep(time.Second)
}

type service1 struct{}

func (s *service1) Name() string {
	return "Service 1"
}

func (s *service1) Start(ctx context.Context) error {
	fmt.Println("Service 1 starting")
	sleep()
	fmt.Println("Service 1 started")
	return nil
}

func (s *service1) Stop(ctx context.Context) error {
	fmt.Println("Service 1 stopping")
	sleep()
	fmt.Println("Service 1 stopped")
	return nil
}

type service2 struct{}

func (s *service2) Name() string {
	return "Service 2"
}

func (s *service2) Start(ctx context.Context) error {
	fmt.Println("Service 2 starting")
	sleep()
	fmt.Println("Service 2 started")
	return nil
}

func (s *service2) Stop(ctx context.Context) error {
	fmt.Println("Service 2 stopping")
	sleep()
	fmt.Println("Service 2 stopped")
	return nil
}

type server struct {
	name       string
	ch         chan struct{}
	cancelFunc context.CancelFunc
}

func (s *server) Name() string {
	return s.Name()
}

func (s *server) Listen(ctx context.Context) error {
	fmt.Println(s.name, " listening starting")
	s.ch = make(chan struct{})
	c, cancelFunc := context.WithCancel(ctx)
	s.cancelFunc = cancelFunc
	select {
	case <-s.ch:
	case <-c.Done():
		fmt.Println(s.name, " listening cancelled")
		return c.Err()
	}
	fmt.Println(s.name, " listening OK")
	return nil
}

func (s *server) Close(ctx context.Context) error {
	s.cancelFunc()
	close(s.ch)
	fmt.Println(s.name, " closing")
	sleep()
	fmt.Println(s.name, " closed")
	return nil
}

func main() {
	ctx := context.Background()

	runner := services.NewRunner()
	defer runner.Finish(ctx)

	err := runner.Run(ctx, &service1{}, &service2{})
	if err != nil {
		panic(err)
	}

	fmt.Println("[hit Ctrl+C] to finish ...")
	err = runner.Run(ctx, &server{name: "Server A"}, &server{name: "Server B"})
	if err != nil {
		panic(err)
	}
}
