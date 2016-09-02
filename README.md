GAMI
====

GO - Asterisk AMI Interface

communicate with the  Asterisk AMI, Actions and Events.

Example connecting to Asterisk and Send Action get Events.

``` go
package main

import (
	"log"

	"github.com/party79/gami"
	"github.com/party79/gami/event"
)

func main() {
	ami, err := gami.Dial("127.0.0.1:5038")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	ami.Run()
	defer ami.Close()
	//install manager
	go func() {
		for {
			select {
			//handle network errors
			case err := <-ami.NetError:
				log.Println("Network Error:", err)
				//try new connection every second
				<-time.After(time.Second)
				if err := ami.Reconnect(); err == nil {
					//call start actions
					ami.Action("Events", gami.Params{"EventMask": "on"})
				}
			case err := <-ami.Error:
				log.Println("error:", err)
			//wait events and process
			case ev := <-ami.Events:
				log.Println("Event Detect: %v", *ev)
				//if want type of events
				log.Println("EventType:", event.New(ev))
			}
		}
	}()
	if err := ami.Login("admin", "root"); err != nil {
		log.Fatal(err)
	}
	if rs, err = ami.Action("Ping", nil); err == nil {
		log.Fatal(rs)
	}
	//or with can do async
	pingResp, pingErr := ami.AsyncAction("Ping", gami.Params{"ActionID": "miping"})
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	if rs, err = ami.Action("Events", ami.Params{"EventMask":"on"}); err != nil {
		fmt.Print(err)
	}
	log.Println("future ping:", <-pingResp)
}
```

###TLS SUPPORT

In order to use TLS connection to manager interface you could `Dial` with additional parameters
``` go
//without TLS
ami, err := gami.Dial("127.0.0.1:5038")

//if certificate is trusted
ami, err := gami.Dial("127.0.0.1:5039", gami.UseTLS)

//if self signed certificate
ami, err := gami.Dial("127.0.0.1:5039", gami.UseTLS, gami.UnsecureTLS)

//if custom tls configuration
ami, err := gami.Dial("127.0.0.1:5039", gami.UseTLSConfig(&tls.Config{}))
```
**WARNING:**
*Only Asterisk >=1.6 supports TLS connection to AMI and
it needs additional configuration(follow the [Asterisk AMI configuration](http://www.asteriskdocs.org/en/3rd_Edition/asterisk-book-html-chunk/AMI-configuration.html) documentation)*

###CUSTOM CONNECTION

``` go
package main

import (
	"log"
	"net"

	"github.com/party79/gami"
)

func main() {
	c, err := net.Dial("tcp", "127.0.0.1:5038")
	if err != nil {
		log.Fatal(err)
	}
	ami, err := gami.NewClient(c)
	if err != nil {
		log.Fatal(err)
	}
	ami.Run()
	defer ami.Close()
	//do something
}
```


CURRENT EVENT TYPES
====

The events use documentation and struct from *PAMI*.

use **party79/gami/event.New()** for get this struct from raw event

EVENT ID           | TYPE TEST  
------------------ | ---------- 
*Newchannel*       | YES
*Newexten*         | YES
*Newstate*         | YES 
*Dial*             | YES 
*ExtensionStatus*  | YES 
*Hangup*           | YES 
*PeerStatus*       | YES
*PeerEntry*	       | YES
*VarSet*           | YES 
*AgentLogin*       | YES
*Agents*           | YES
*AgentLogoff*      | YES
*AgentConnect*     | YES
*RTPReceiverStats* | YES
*RTPSenderStats*   | YES
*Bridge*           | YES
