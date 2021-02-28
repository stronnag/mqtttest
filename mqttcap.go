package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"io/ioutil"
	"strconv"
	"syscall"
	"time"
	"net/url"
	"math/rand"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func NewTlsConfig(cafile string) (*tls.Config, string) {
	if len(cafile) == 0 {
		return nil, "tcp"
	} else {
		fmt.Fprintf(os.Stderr, "Use CAfile %s\n", cafile)
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

func main() {
	var wfh *os.File
	var broker string
	var topic string
	var port int
	var cafile string
	var user string
	var passwd string
	var mqttopts string

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s -broker URI [options] ...\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.StringVar(&mqttopts, "broker", "", "Mqtt URI (mqtt://[user[:pass]@]broker[:port]/topic[?cafile=file])")
	logdir := flag.String("logdir", "/tmp", "log directory for messages")
	splittime := flag.Int("splittime", 300, "split time for logs")

	flag.Parse()

	if mqttopts == "" {
		fmt.Fprintln(os.Stderr, "need -broker option")
		os.Exit(255)
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

	rand.Seed(time.Now().UnixNano())
	clientid := fmt.Sprintf("%x", rand.Int63())
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
	hpath := fmt.Sprintf("%s://%s:%d%s", scheme, broker, port, mpath)

	opts.AddBroker(hpath)
	opts.SetClientID(clientid)
	opts.SetCleanSession(true)

	if user != "" {
		opts.SetUsername(user)
		if passwd != "" {
			opts.SetPassword(passwd)
		}
	}

	var lt time.Time

	opts.SetTLSConfig(tlsconf)
	opts.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe(topic, byte(0), func(c mqtt.Client, m mqtt.Message) {
			now := time.Now()
			et := now.UnixNano() / 1000000
			if time.Since(lt) > time.Duration(time.Duration(*splittime)*time.Second) {
				if wfh != nil {
					wfh.Close()
				}
				fname := fmt.Sprintf("%s/%s.txt", *logdir, now.Format("20060102150405"))
				var err error
				if wfh, err = os.Create(fname); err != nil {
					log.Fatal(err)
				}
				s := fmt.Sprintf("%s|Connected to %s - %s", et, broker, topic)
				fmt.Fprintln(wfh, s)
				fmt.Fprintln(os.Stderr, s)
			}
			lt = now
			s := fmt.Sprintf("%d|%s", et, string(m.Payload()))
			fmt.Fprintln(wfh, s)
			fmt.Fprintln(os.Stderr, s)
		}); token.Wait() && token.Error() != nil {
			log.Fatal(token.Error())
		}
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	<-c
	if wfh != nil {
		wfh.Close()
	}
}
