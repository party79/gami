package gami

import (
	"net/textproto"
	"sync"
	"testing"
	"time"

	"github.com/bit4bit/gami/mock"
)

func TestLogin(t *testing.T) {
	srv := amimock.NewAmiServer()
	defer srv.Close()
	ami, err := Dial(srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go ami.Run()
	defer ami.Close()
	closech := defaultInstaller(t, ami)
	defer func() {
		closech <- true
	}()

	//example mocking login of asterisk
	srv.Mock("Login", func(params textproto.MIMEHeader) map[string]string {
		return map[string]string{
			"Response": "OK",
			"ActionID": params.Get("Actionid"),
		}
	})
	ami.Login("admin", "admin")
}

func TestMultiAsyncActions(t *testing.T) {
	srv := amimock.NewAmiServer()
	defer srv.Close()
	ami, err := Dial(srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go ami.Run()
	defer ami.Close()
	closech := defaultInstaller(t, ami)
	defer func() {
		closech <- true
	}()

	tests := 10
	workers := 5

	wg := &sync.WaitGroup{}
	for ti := tests; ti > 0; ti-- {
		resWorkers := make(chan (<-chan *AMIResponse), workers)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				chres, err := ami.AsyncAction("Test", nil)
				if err != nil {
					t.Error(err)
				}
				resWorkers <- chres
				wg.Done()
			}()
		}
		go func() {
			wg.Wait()
			close(resWorkers)
		}()

		for resp := range resWorkers {
			select {
			case <-time.After(time.Second * 5):
				t.Fatal("asyncAction locked")
			case <-resp:
			}
		}

	}

}

func defaultInstaller(t *testing.T, ami *AMIClient) chan bool {
	var closech chan bool = make(chan bool, 1)
	go func() {
		for {
			select {
			//handle network errors
			case err := <-ami.NetError:
				t.Error("Network Error:", err)
			case err := <-ami.Error:
				t.Error("error:", err)
			//wait events and process
			case <-ami.Events:
				//t.Log("Event:", *ev)
			case <-closech:
				return
			}
		}
	}()
	return closech
}
