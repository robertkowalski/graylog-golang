package gelf

import (
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/lintianzhi/graylogd"
)

var validJson = `{
			"version": "1.0",
			"host": "localhost",
			"timestamp": "123312312",
			"facility": "Google Go",
			"short_message": "Hello From Golang! :)"
	}`

var inValidJson = `{
			"_id": "23",
			"version": "1.0",
			"host": "localhost",
			"timestamp": "123312312",
			"facility": "Google Go",
			"short_message": "Hello From Golang! :)"
	}`

func Benchmark_LogWithShortMessage(b *testing.B) {
	b.StopTimer()
	g := New(Config{})

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		g.Log("Hello World")
	}
}

func Benchmark_LogWithChunks(b *testing.B) {
	b.StopTimer()
	g := New(Config{
		MaxChunkSizeWan: 10,
		MaxChunkSizeLan: 10,
	})

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		g.Log("sdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddfsdfsdsdfdsfdsddddf")
	}
}

func Test_New_itShouldUseDefaultConfigValuesIfNoOtherProvided(t *testing.T) {
	g := New(Config{})

	assert.Equal(t, g.Config.GraylogPort, defaultGraylogPort)
	assert.Equal(t, g.Config.GraylogHostname, defaultGraylogHostname)
	assert.Equal(t, g.Config.Connection, defaultConnection)
	assert.Equal(t, g.Config.MaxChunkSizeWan, defaultMaxChunkSizeWan)
	assert.Equal(t, g.Config.MaxChunkSizeLan, defaultMaxChunkSizeLan)
}

func Test_New_itShouldUseConfigValuesFromArguments(t *testing.T) {
	g := New(Config{
		GraylogPort:     80,
		GraylogHostname: "foobarhost",
		Connection:      "wlan",
		MaxChunkSizeWan: 42,
		MaxChunkSizeLan: 1337,
	})

	assert.Equal(t, g.Config.GraylogPort, 80)
	assert.Equal(t, g.Config.GraylogHostname, "foobarhost")
	assert.Equal(t, g.Config.Connection, "wlan")
	assert.Equal(t, g.Config.MaxChunkSizeWan, 42)
	assert.Equal(t, g.Config.MaxChunkSizeLan, 1337)
}

func Test_ParseJson_itShouldReturnTypeMapStringInterface(t *testing.T) {
	g := New(Config{})
	res := g.ParseJson(validJson)

	assert.Equal(t, reflect.TypeOf(res), reflect.TypeOf(make(map[string]interface{})))
}

func Test_ParseJson_itShouldParseTheStringToJson(t *testing.T) {
	g := New(Config{})
	res := g.ParseJson(validJson)

	assert.Equal(t, res["version"], "1.0")
	assert.Equal(t, res["host"], "localhost")
	assert.Equal(t, res["timestamp"], "123312312")
	assert.Equal(t, res["facility"], "Google Go")
	assert.Equal(t, res["short_message"], "Hello From Golang! :)")
}

func Test_TestForForbiddenValues_itShouldReturnAnErrorIfForbiddenValuesAppear(t *testing.T) {
	g := New(Config{})
	res := g.ParseJson(inValidJson)
	err := g.TestForForbiddenValues(res)

	assert.NotEqual(t, nil, err)
}

func Test_TestSend_itShouldSendUdpPacketsToAServer(t *testing.T) {
	g := New(Config{
		GraylogPort: 55555,
	})

	done := make(chan int)
	go Server(done, 55555, t)
	g.Send([]byte("Hello Graylog"))
	<-done
}

func Test_IntToBytes_itShouldCreateBytesFromInts(t *testing.T) {
	g := New(Config{})

	res := g.IntToBytes(20)
	expected := make([]int32, 1)
	expected[0] = 20

	assert.Equal(t, bytes.Runes(res), expected)
}

func Test_GetChunksize_itShouldReturnTheValuesForWan(t *testing.T) {
	g := New(Config{
		Connection:      "wan",
		MaxChunkSizeWan: 42,
		MaxChunkSizeLan: 1337,
	})

	res := g.GetChunksize()

	assert.Equal(t, 42, res)
}

func Test_GetChunksize_itShouldReturnTheValuesForLan(t *testing.T) {
	g := New(Config{
		Connection:      "lan",
		MaxChunkSizeWan: 42,
		MaxChunkSizeLan: 1337,
	})

	res := g.GetChunksize()

	assert.Equal(t, 1337, res)
}

func Test_CreateChunkedMessages_itShouldStartWithTheMagicNumber(t *testing.T) {
	g := New(Config{})
	b := []byte("message")
	buffer := bytes.NewBuffer(b)

	packet := g.CreateChunkedMessage(1, 0, []byte("id"), buffer)
	res := packet.String()

	assert.Equal(t, strings.Contains(res, "\x1e\x0f"), true)
}

func Test_ChunkSize(t *testing.T) {

	waitChan := make(chan bool, 1)
	var realB []byte
	daeCfg := graylogd.Config{
		ListenAddr: "127.0.0.1:2211",
		HandleRaw: func(b []byte) {
			assert.Equal(t, realB, b)
			waitChan <- true
		},
		HandleError: func(addr *net.UDPAddr, err error) {
			t.Fatal("should be no error", err)
		},
	}
	logd, err := graylogd.NewGraylogd(daeCfg)
	assert.Equal(t, nil, err)
	assert.Equal(t, nil, logd.Run())
	defer logd.Close()

	client := New(Config{
		GraylogPort:     2211,
		GraylogHostname: "127.0.0.1",
		MaxChunkSizeWan: 1,
		MaxChunkSizeLan: 1,
	})

	msgs := []string{
		"11111",
		"123jjdd",
	}
	for _, msg := range msgs {

		realB = []byte(msg)

		client.Log(msg)
		select {
		case <-waitChan:
		case <-time.After(time.Second):
			t.Fatal("message is not received")
		}
	}
}

func Test_CreateChunkedMessages_itShouldContainAnId(t *testing.T) {
	g := New(Config{})
	b := []byte("message")
	buffer := bytes.NewBuffer(b)

	packet := g.CreateChunkedMessage(1, 0, []byte("myId"), buffer)
	res := packet.String()

	assert.Equal(t, strings.Contains(res, "myId"), true)
}

func Test_CreateChunkedMessages_itShouldHaveTheIndex(t *testing.T) {
	g := New(Config{})
	b := []byte("message")
	buffer := bytes.NewBuffer(b)

	packet := g.CreateChunkedMessage(13, 42, []byte("id"), buffer)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int8(13))

	assert.Equal(t, bytes.Contains(packet.Bytes(), buf.Bytes()), true)
}

func Test_CreateChunkedMessages_itShouldHaveThePacketCount(t *testing.T) {
	g := New(Config{})
	b := []byte("message")
	buffer := bytes.NewBuffer(b)

	packet := g.CreateChunkedMessage(133, 42, []byte("id"), buffer)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int8(42))

	assert.Equal(t, bytes.Contains(packet.Bytes(), buf.Bytes()), true)
}

func Server(done chan<- int, port int, t *testing.T) {
	laddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	buffer := make([]byte, 1024)
	for {
		conn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			panic(err)
		}

		for {
			n, err := conn.Read(buffer)
			if err != nil {
				panic(err)
			}
			conn.Close()
			if string(buffer[:n]) != "Hello Graylog" {
				t.Error("TestServer Error - String not Equal.")
			}
			done <- 0
			return
		}

		conn.Close()
	}
}
