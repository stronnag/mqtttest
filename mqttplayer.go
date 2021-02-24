package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"fmt"
	"crypto/tls"
	"crypto/x509"
	"time"
	"strings"
	"io/ioutil"
	"math/rand"
	"os"
	"flag"
	"log"
	"strconv"
	"bufio"
	"net/url"
)

func NewTlsConfig(cafile string) (*tls.Config, string) {
	if len(cafile) == 0 {
		return nil, "tcp"
	} else {
		certpool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(cafile)
		if err != nil {
			log.Fatalln(err.Error())
		}
		certpool.AppendCertsFromPEM(ca)
		return &tls.Config{
			RootCAs:            certpool,
			InsecureSkipVerify: true, ClientAuth: tls.NoClientCert,
		},
			"ssl"
	}
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v\n", err)
}

var qos = flag.Int("qos", 0, "The QoS for message publication")

func publish(client mqtt.Client, topic string, msg string) {
	token := client.Publish(topic, byte(*qos), false, msg)
	token.Wait()
}

func main() {
	var broker string
	var topic string
	var port int
	var mqttopts string
	var cafile string
	var user string
	var passwd string

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s -broker URI [-qos qos] file\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.StringVar(&mqttopts, "broker", "", "Mqtt URI (mqtt://[user[:pass]@]broker[:port]/topic[?cafile=file])")

	flag.Parse()
	files := flag.Args()
	if len(files) < 1 {
		fmt.Fprintln(os.Stderr, "Need a log file")
		return
	}

	u, err := url.Parse(mqttopts)
	if err != nil {
		log.Fatal(err)
	}

	if len(u.Path) > 0 {
		topic = u.Path[1:]
	}
	port, _ = strconv.Atoi(u.Port())
	broker = u.Hostname()

	up := u.User
	user = up.Username()
	passwd, _ = up.Password()

	q := u.Query()
	ca := q["cafile"]
	if len(ca) > 0 {
		cafile = ca[0]
	}
	if broker == "" || topic == "" {
		fmt.Fprintln(os.Stderr, "need broker and topic")
		return
	}

	if port == 0 {
		port = 1883
	}

	tlsconf, scheme := NewTlsConfig(cafile)
	if u.Scheme == "ws" {
		scheme = "ws"
	}

	if u.Scheme == "wss" {
		tlsconf = &tls.Config{RootCAs: nil, ClientAuth: tls.NoClientCert}
		scheme = "wss"
	}

	if tlsconf == nil && (u.Scheme == "mqtts" || u.Scheme == "ssl") {
		tlsconf = &tls.Config{RootCAs: nil, ClientAuth: tls.NoClientCert}
		scheme = "ssl"
	}

	if len(os.Getenv("NOVERIFYSSL")) > 0 && tlsconf != nil {
		tlsconf.InsecureSkipVerify = true
	}

	opts := mqtt.NewClientOptions()
	mpath := ""
	if scheme == "ws" || scheme == "wss" {
		mpath = "/mqtt"
	}

	rand.Seed(time.Now().UnixNano())
	clientid := fmt.Sprintf("%x", rand.Int63())

	hpath := fmt.Sprintf("%s://%s:%d%s", scheme, broker, port, mpath)
	opts.AddBroker(hpath)
	opts.SetTLSConfig(tlsconf)
	opts.SetClientID(clientid)
	opts.SetUsername(user)
	opts.SetPassword(passwd)
	opts.SetDefaultPublishHandler(messagePubHandler)

	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	rfh, err := os.Open(files[0])
	if err == nil {
		lastt := 0.0
		defer rfh.Close()
		scanner := bufio.NewScanner(rfh)
		for scanner.Scan() {
			line := scanner.Text()
			if parts := strings.Split(line, "\t"); len(parts) == 2 {
				toff, _ := strconv.ParseFloat(parts[0], 64)
				publish(client, topic, parts[1])
				if lastt != 0.0 {
					tdiff := int64(1000.0 * (toff - lastt))
					time.Sleep(time.Duration(tdiff) * time.Millisecond)
				}
				lastt = toff
			} else if parts := strings.Split(line, "|"); len(parts) == 2 {
				tint, _ := strconv.ParseInt(parts[0], 10, 64)
				toff := float64(tint) / 1000.0
				publish(client, topic, parts[1])
				if lastt != 0.0 {
					tdiff := int64(1000.0 * (toff - lastt))
					time.Sleep(time.Duration(tdiff) * time.Millisecond)
				}
				lastt = toff
			} else {
				publish(client, topic, line)
				if !strings.HasPrefix(line, "wpno") {
					time.Sleep(1 * time.Second)
				}
			}
		}
	} else {
		log.Fatal(err)
	}
}
