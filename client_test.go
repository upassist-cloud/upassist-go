package upassist

import "testing"

func TestNewClient(t *testing.T) {
	client, err := NewClient("https://api.upassist.cloud")
	if err != nil {
		t.Fatal(err)
	}
	if client.Server != "https://api.upassist.cloud/" {
		t.Fatalf("server = %s", client.Server)
	}
}
