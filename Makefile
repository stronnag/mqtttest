# Build statically to avoid libc version issues
LDFLAGS = -s -w -extldflags -static

all : mqttsub mqttplayer mqttcap

% : %.go
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $@  $<

clean:
	rm -f  mqttsub mqttplayer mqttcap
