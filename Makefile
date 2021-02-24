# Build statically to avoid libc version issues
LDFLAGS = -s -w -extldflags -static

all : mqttsub mqttplayer

mqttsub : mqttsub.go
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o  mqttsub  mqttsub.go

mqttplayer : mqttplayer.go
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o  mqttplayer  mqttplayer.go

clean:
	rm -f  mqttsub mqttplayer
