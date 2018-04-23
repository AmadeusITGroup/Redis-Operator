package fake

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// RedisServer Fake Redis Server struct
type RedisServer struct {
	sync.Mutex
	Ln        net.Listener
	Responses map[string][]interface{}
	test      *testing.T
}

// NewRedisServer returns new Fake RedisServer instance
func NewRedisServer(t *testing.T) *RedisServer {
	ln, err := net.Listen("tcp", "localhost:0")

	if err != nil {
		t.Fatal("Unable to create a FakeRedisServer, err:", err)
		return nil
	}

	var srv = &RedisServer{
		Responses: make(map[string][]interface{}),
		Ln:        ln,
		test:      t,
	}

	go srv.handleConnection()

	return srv
}

// Close possible ressources
func (r *RedisServer) Close() {
	r.Ln.Close()
}

// GetHostPort return the host port of redis server
func (r *RedisServer) GetHostPort() string {
	return r.Ln.Addr().String()
}

func (r *RedisServer) popResponse(rq string) (interface{}, bool) {
	r.Lock()
	defer r.Unlock()
	list, ok := r.Responses[rq]
	if ok {
		if len(list) != 0 {
			val, newlist := list[0], list[1:]
			r.Responses[rq] = newlist
			return val, true
		}
	}
	return fmt.Errorf("fake.RedisNode: cannot map request '%s' to registered response", rq), false
}

// PushResponse add a response to a specific request
func (r *RedisServer) PushResponse(rq string, response interface{}) {
	r.Lock()
	defer r.Unlock()
	list, ok := r.Responses[rq]
	if !ok {
		list = []interface{}{}
	}
	r.Responses[rq] = append(list, response)
}

func (r *RedisServer) handleConnection() {
	buf := make([]byte, 4096)
	for {
		conn, err := r.Ln.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			} else {
				break
			}
		}
		defer conn.Close()
		var wait sync.WaitGroup
		wait.Add(1)
		go func(conn net.Conn, wait *sync.WaitGroup) {
			for {
				// TODO handle buffer full case
				n, err := conn.Read(buf)
				if err != nil || n == 0 {
					break
				}
				rqs := cleanCommand(buf[0:n])
				for _, rq := range rqs {
					resp, _ := r.popResponse(rq)
					if err = writeResponse(resp, conn); err != nil || n == 0 {
						break
					}
				}
			}
			wait.Done()
		}(conn, &wait)
		wait.Wait()
	}
}

// does not handle arrays of arrays
func cleanCommand(buf []byte) []string {
	commands := []string{}
	sliceCommand := []string{}

	reader := bytes.NewReader(buf)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "*") {
			if len(sliceCommand) != 0 {
				// new command starting
				commands = append(commands, strings.Join(sliceCommand, " "))
				sliceCommand = []string{}
			}
			continue
		}
		if strings.HasPrefix(scanner.Text(), "$") {
			// next string size
			continue
		}
		// else string
		sliceCommand = append(sliceCommand, scanner.Text())
	}
	commands = append(commands, strings.Join(sliceCommand, " "))

	return commands
}

func writeResponse(response interface{}, conn io.Writer) error {
	writeBuf := bytes.NewBuffer(make([]byte, 0, 128))
	writeScratch := make([]byte, 0, 128)

	if _, err := writeTo(writeBuf, writeScratch, response, false); err != nil {
		return err
	}

	if _, err := writeBuf.WriteTo(conn); err != nil {
		return err
	}

	return nil
}

var (
	delim = []byte{'\r', '\n'}

	errPrefix     = []byte{'-'}
	intPrefix     = []byte{':'}
	bulkStrPrefix = []byte{'$'}
	arrayPrefix   = []byte{'*'}
	nilFormatted  = []byte("$-1\r\n")
)

func writeArrayHeader(w io.Writer, buf []byte, l int64) (int64, error) {
	buf = strconv.AppendInt(buf[:0], l, 10)
	var err error
	var written int64
	written, err = writeBytesHelper(w, arrayPrefix, written, err)
	written, err = writeBytesHelper(w, buf, written, err)
	written, err = writeBytesHelper(w, delim, written, err)
	return written, err
}

func writeBytesHelper(w io.Writer, b []byte, lastWritten int64, lastErr error) (int64, error) {
	if lastErr != nil {
		return lastWritten, lastErr
	}
	i, err := w.Write(b)
	return int64(i) + lastWritten, err
}

// takes in something, m, and encodes it and writes it to w. buf is used as a
// pre-alloated byte buffer for encoding integers (expected to have a length of
// 0), so we don't have to re-allocate a new one every time we convert an
// integer to a string.
// noArrayHeader means don't write out the headers to any arrays, just
// inline all the elements in the array
func writeTo(w io.Writer, buf []byte, m interface{}, noArrayHeader bool) (int64, error) {
	//fmt.Printf("%T\n", m)
	switch mt := m.(type) {
	case []byte:
		return writeStr(w, buf, mt)
	case string:
		var sbuf []byte
		sbuf, buf = stringSlicer(buf, mt)
		return writeStr(w, buf, sbuf)
	case bool:
		buf = buf[:0]
		if mt {
			buf = append(buf, '1')
		} else {
			buf = append(buf, '0')
		}
		return writeStr(w, buf[1:], buf[:1])
	case nil:
		return writeNil(w)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		i := anyIntToInt64(mt)
		return writeInt(w, buf, i)
	case float32:
		return writeFloat(w, buf, float64(mt), 32)
	case float64:
		return writeFloat(w, buf, mt, 64)
	case error:
		return writeErr(w, buf, mt)
	// specific structs that cannot be represented with simple combinations of other types
	case ClusterSlotsSlot:
		l := 2 + len(mt.Nodes)
		var totalWritten int64
		if !noArrayHeader {
			written, err := writeArrayHeader(w, buf, int64(l))
			totalWritten += written
			if err != nil {
				return totalWritten, err
			}
		}
		written, err := writeTo(w, buf, mt.Min, noArrayHeader)
		totalWritten += written
		if err != nil {
			return totalWritten, err
		}
		written, err = writeTo(w, buf, mt.Max, noArrayHeader)
		totalWritten += written
		if err != nil {
			return totalWritten, err
		}
		for _, node := range mt.Nodes {
			written, err = writeTo(w, buf, node, noArrayHeader)
			totalWritten += written
			if err != nil {
				return totalWritten, err
			}
		}
		return totalWritten, nil
	default:
		// Fallback to reflect-based.
		switch reflect.TypeOf(m).Kind() {
		case reflect.Slice:
			rm := reflect.ValueOf(mt)
			l := rm.Len()
			var totalWritten, written int64
			var err error
			if !noArrayHeader {
				written, err = writeArrayHeader(w, buf, int64(l))
				totalWritten += written
				if err != nil {
					return totalWritten, err
				}
			}
			for i := 0; i < l; i++ {
				vv := rm.Index(i).Interface()
				written, err = writeTo(w, buf, vv, noArrayHeader)
				totalWritten += written
				if err != nil {
					return totalWritten, err
				}
			}
			return totalWritten, nil
		case reflect.Struct:
			rm := reflect.ValueOf(mt)
			l := rm.NumField()
			var totalWritten, written int64
			var err error
			if (l != 1) && !noArrayHeader {
				written, err = writeArrayHeader(w, buf, int64(l))
				totalWritten += written
				if err != nil {
					return totalWritten, err
				}
			}
			for i := 0; i < rm.NumField(); i++ {
				vv := rm.Field(i).Interface()
				written, err = writeTo(w, buf, vv, noArrayHeader)
				totalWritten += written
				if err != nil {
					return totalWritten, err
				}
			}
			return totalWritten, nil
		default:
			return writeStr(w, buf, []byte(fmt.Sprint(m)))
		}
	}
}

func writeStr(w io.Writer, buf, b []byte) (int64, error) {
	var err error
	var written int64
	buf = strconv.AppendInt(buf[:0], int64(len(b)), 10)
	written, err = writeBytesHelper(w, bulkStrPrefix, written, err)
	written, err = writeBytesHelper(w, buf, written, err)
	written, err = writeBytesHelper(w, delim, written, err)
	written, err = writeBytesHelper(w, b, written, err)
	written, err = writeBytesHelper(w, delim, written, err)
	return written, err
}
func writeErr(w io.Writer, buf []byte, ierr error) (int64, error) {
	var err error
	var written int64
	written, err = writeBytesHelper(w, errPrefix, written, err)
	written, err = writeBytesHelper(w, []byte(ierr.Error()), written, err)
	written, err = writeBytesHelper(w, delim, written, err)
	return written, err
}
func writeInt(
	w io.Writer, buf []byte, i int64) (int64, error) {
	buf = strconv.AppendInt(buf[:0], i, 10)
	var err error
	var written int64
	written, err = writeBytesHelper(w, intPrefix, written, err)
	written, err = writeBytesHelper(w, buf, written, err)
	written, err = writeBytesHelper(w, delim, written, err)
	return written, err
}
func writeFloat(w io.Writer, buf []byte, f float64, bits int) (int64, error) {
	buf = strconv.AppendFloat(buf[:0], f, 'f', -1, bits)
	return writeStr(w, buf[len(buf):], buf)
}
func writeNil(w io.Writer) (int64, error) {
	written, err := w.Write(nilFormatted)
	return int64(written), err
}

// Given a preallocated byte buffer and a string, this will copy the string's
// contents into buf starting at index 0, and returns two slices from buf: The
// first is a slice of the string data, the second is a slice of the "rest" of
// buf following the first slice
func stringSlicer(buf []byte, s string) ([]byte, []byte) {
	sbuf := append(buf[:0], s...)
	return sbuf, sbuf[len(sbuf):]
}

func anyIntToInt64(m interface{}) int64 {
	switch mt := m.(type) {
	case int:
		return int64(mt)
	case int8:
		return int64(mt)
	case int16:
		return int64(mt)
	case int32:
		return int64(mt)
	case int64:
		return mt
	case uint:
		return int64(mt)
	case uint8:
		return int64(mt)
	case uint16:
		return int64(mt)
	case uint32:
		return int64(mt)
	case uint64:
		return int64(mt)
	}
	panic(fmt.Sprintf("anyIntToInt64 got bad arg: %#v", m))
}
