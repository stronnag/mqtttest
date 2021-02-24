# Simple MQTT tools

Two applications are provided primarily to test the usage of the BulletGCSS / MQTT telemetry protocol.

* **mqttsub** Subscribe to a MQTT service, display received data, optionally generate time-stamped log
* **mqttplayer** Replay a MQTT log (from `mqttsub`) to a MQTT server, preserving message timing.

```
$ mqttsub --help
Usage of mqttsub -broker URI [options] ...
  -broker string
    	Mqtt URI (mqtt://[user[:pass]@]broker[:port]/topic[?cafile=file])
  -clientid string
    	A clientid for the connection (default "eeyore30")
  -log string
    	log file for messages
  -qos int
    	The QoS to subscribe to messages at
```

```
$ mqttplayer --help
Usage of mqttplayer -broker URI [-qos qos] file
  -broker string
    	Mqtt URI (mqtt://[user[:pass]@]broker[:port]/topic[?cafile=file])
  -qos int
    	The QoS for message publication
```

Note that the scheme (**mqtt**:// in the `--help` text) is interpreted as:

* ws - Websocket (vice TCP socket), ensure the websocket port is also specificed
* wss - Encrypted websocket, ensure the TLS websocket port is also specificed. TLS validation is performed using the system CA files.
* mqtts - Secure (TLS) TCP connection. Ensure the TLS port is specified. TLS validation is performed using the system CA files.
* mqtt (or anyother scheme) - TCP connection. If `?cafile=file` is specified, then that is used for TLS validation (and the TLS port should be specified).

## Log file

The log file contains payload data, prefixed with a time_t-like timestamp with millsecond precision. The timestamp and payload are separated by a TAB character.

```
1611845402.129  cs:JRandomUAV,
1611845402.130  flt:0,ont:60,ran:20,pan:-10,hea:263,ggc:200,alt:0,asl:33,gsp:0,bpv:16.80,cad:0,cud:0.11,rsi:100,
```

`mqttplayer` can also replay BulletGCSS log files. These are characterised by a millisecond Unix epoch timestamp, a '|' spearator followed by the data.

if the log contains no timestamp and separator, the contents are replayed at 1 second interval.
