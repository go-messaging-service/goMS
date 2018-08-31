package commonServices

import (
	"bufio"
	testUtils "goms-server/src/domain/services/test"
	"net"
	"testing"
)

var client, server *net.Conn
var serverReader *bufio.Reader

const TEST_STRING string = "test123"

func TestSendStringWorks(t *testing.T) {
	client, _, server, serverReader = testUtils.InitPipe()

	if serverReader.Buffered() != 0 {
		t.Error("Buffered bytes must be 0")
		t.Fail()
	}

	go func(conn *net.Conn) { SendStringTo(conn, TEST_STRING) }(client)

	data, _, err := serverReader.ReadLine()

	if err != nil {
		t.Fail()
	}

	if string(data) != TEST_STRING {
		t.Error("Buffered bytes must be !=0")
		t.Fail()
	}
}