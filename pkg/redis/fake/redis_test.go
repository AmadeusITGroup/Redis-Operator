package fake

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestCleanCommand(t *testing.T) {
	testCases := []struct {
		request  []byte
		response []string
	}{
		{
			request: []byte(`*2
*4
$7
CLUSTER
$4
MEET
$9
127.0.0.1
$4
6667
*4
$7
CLUSTER
$4
MEET
$9
127.0.0.2
$4
6668`),
			response: []string{"CLUSTER MEET 127.0.0.1 6667", "CLUSTER MEET 127.0.0.2 6668"},
		},
	}

	for _, tt := range testCases {
		out := cleanCommand(tt.request)
		if !reflect.DeepEqual(out, tt.response) {
			t.Errorf("cleanCommand bad cleaning, expected %s, got %s", tt.response, out)
		}
	}
}

func TestWriteTo(t *testing.T) {
	testCases := []struct {
		input  interface{}
		output []byte
	}{
		{input: []byte(`test`), output: []byte("$4\r\ntest\r\n")},
		{input: 42, output: []byte(":42\r\n")},
		{input: int8(42), output: []byte(":42\r\n")},
		{input: int16(42), output: []byte(":42\r\n")},
		{input: int64(42), output: []byte(":42\r\n")},
		{input: int32(42), output: []byte(":42\r\n")},
		{input: uint(42), output: []byte(":42\r\n")},
		{input: uint8(42), output: []byte(":42\r\n")},
		{input: uint16(42), output: []byte(":42\r\n")},
		{input: uint32(42), output: []byte(":42\r\n")},
		{input: uint64(42), output: []byte(":42\r\n")},
		{input: float32(42), output: []byte("$2\r\n42\r\n")},
		{input: float64(42), output: []byte("$2\r\n42\r\n")},
		{input: nil, output: nilFormatted},
		{input: true, output: []byte("$1\r\n1\r\n")},
		{input: false, output: []byte("$1\r\n0\r\n")},
		{input: errors.New("error"), output: []byte("-error\r\n")},
		{input: "test", output: []byte("$4\r\ntest\r\n")},
		{input: []string{"test", "test2"}, output: []byte("*2\r\n$4\r\ntest\r\n$5\r\ntest2\r\n")},
		{
			input: []ClusterSlotsSlot{
				{
					Min: 0,
					Max: 100,
					Nodes: []ClusterSlotsNode{
						{IP: "1.2.3.4", Port: 1234},
						{IP: "1.2.3.5", Port: 1235},
					},
				},
			},
			output: []byte("*1\r\n*4\r\n:0\r\n:100\r\n*2\r\n$7\r\n1.2.3.4\r\n:1234\r\n*2\r\n$7\r\n1.2.3.5\r\n:1235\r\n"),
		},
	}

	for _, tt := range testCases {
		buf := make([]byte, 4096)
		writter := new(bytes.Buffer)
		size, err := writeTo(writter, buf, tt.input, false)
		if err != nil {
			t.Errorf("unexpected error on writeTo(%s): %v", tt.input, err)
		}
		if size != int64(len(tt.output)) || !reflect.DeepEqual(tt.output, writter.Bytes()) {
			t.Errorf("unexpected data written on write(%s), expected '%s' (size:%d), got '%s' (size:%d)", tt.input, tt.output, int64(len(tt.output)), writter.Bytes(), size)
		}
	}
}

func TestNewRedisServer(t *testing.T) {
	server := NewRedisServer(t)
	defer server.Close()
	_, _, err := net.SplitHostPort(server.GetHostPort())
	if err != nil {
		t.Errorf("Wrong HostPort format: %s", server.GetHostPort())
	}

	server.PushResponse("REQUEST", "RESP1")
	server.PushResponse("REQUEST", "RESP2")

	conn, err := net.Dial("tcp", server.GetHostPort())
	if err != nil {
		t.Errorf("Cannot connec to fake redis server: %v", err)
	}
	defer conn.Close()

	testCases := []struct {
		input  string
		output []string
	}{
		{input: "REQUEST", output: []string{"$5\r\n", "RESP1\r\n"}},
		{input: "REQUEST", output: []string{"$5\r\n", "RESP2\r\n"}},
		{input: "REQUEST", output: []string{"-fake.RedisNode: cannot map request 'REQUEST' to registered response\r\n"}},
	}

	for i, tt := range testCases {
		// write to fake redis
		fmt.Fprintf(conn, tt.input)

		//read from fake redis
		var message []string
		reader := bufio.NewReader(conn)

		for range tt.output {
			out, readerr := reader.ReadString('\n')
			if readerr != nil {
				t.Errorf("[test %d] Unexpected error at read: %v", i, readerr)
			}
			message = append(message, out)
		}

		if !reflect.DeepEqual(message, tt.output) {
			t.Errorf("[test %d] Unexpected answer to '%s' expected '%s', got '%s'", i, tt.input, tt.output, message)
		}
	}
}
