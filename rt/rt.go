package rt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.ntppool.org/common/logger"
)

// RT is the client for posting messages to request tracker
type RT struct {
	hclient *http.Client
	config  *rtconfig
}

// AddressQueue contains a Address to Queue mapping
type AddressQueue map[string]string

type rtconfig struct {
	RTUrl  string       `json:"rt-url"`
	Queues AddressQueue `json:"queues"`
}

// New configures a new RT client with the specified configuration file
func New(configfile string) (*RT, error) {
	cfg, err := loadConfig(configfile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration file '%s': %s", configfile, err)
	}

	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	hclient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}

	return &RT{hclient: hclient, config: cfg}, nil
}

func loadConfig(file string) (*rtconfig, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := rtconfig{}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (rt *RT) addressToQueueAction(email string) (string, string) {
	email = strings.ToLower(email)

	idx := strings.Index(email, "@")
	if idx < 1 {
		return "", "correspond"
	}

	local := email[0:idx]

	for _, address := range []string{email, local} {
		for target, queue := range rt.config.Queues {
			// log.Printf("testing address address='%s' target='%s' queue='%s'",
			// 	address, target, queue)

			if address == target {
				return queue, "correspond"
			}
			if idx = strings.Index(target, "@"); idx > 0 {
				target = target[0:idx] + "-comment" + target[idx:]
			} else {
				target = target + "-comment"
			}
			if address == target {
				return queue, "comment"
			}
		}
	}

	return "", "correspond"
}

// Error provides a custom error type for the RT client
type Error struct {
	NotFound bool // Set if the queue wasn't found
	msg      string
}

func (e Error) Error() string {
	if e.NotFound {
		return fmt.Sprintf("%s (notfound=true)", e.msg)
	}
	return fmt.Sprintf("%s", e.msg)
}

// Postmail sends the message to the RT queue matching the specified recipient
func (rt *RT) Postmail(ctx context.Context, recipient string, message string) error {
	log := logger.FromContext(ctx)

	queue, action := rt.addressToQueueAction(recipient)
	if len(queue) == 0 {
		return &Error{
			NotFound: true,
			msg:      fmt.Sprintf("Queue not found for %q (returning 404)", recipient),
		}
	}

	form := url.Values{
		"queue":  []string{queue},
		"action": []string{action},
	}

	log.InfoContext(ctx, "posting to RT queue",
		"queue", queue,
		"action", action,
		"recipient", recipient,
	)

	form.Add("message", message)

	resp, err := rt.hclient.PostForm(
		rt.config.RTUrl,
		form,
	)
	if err != nil {
		return fmt.Errorf("postform err: %s", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading RT response: %s", err)
	}
	resp.Body.Close()

	log.DebugContext(ctx, "RT response",
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	if strings.Contains(string(body), "failure") {
		return fmt.Errorf("RT failure")
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("status code %d (>299)", resp.StatusCode)
	}

	return nil
}
