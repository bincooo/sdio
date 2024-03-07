package sdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RomiChan/websocket"
	"math/rand"
	"net/http"
	"net/url"
)

type JoinCompleted struct {
	Msg     string     `json:"msg"`
	EventId string     `json:"event_id"`
	Success bool       `json:"success"`
	Output  joinOutput `json:"output"`
}

type joinOutput struct {
	Generating      bool    `json:"is_generating"`
	Duration        float64 `json:"duration"`
	AverageDuration float64 `json:"average_duration"`

	Data []interface{} `json:"data"`
}

type Client struct {
	bu      *url.URL
	funcMap map[string]func(j JoinCompleted, data []byte) map[string]interface{}
}

func New(baseUrl string) (*Client, error) {
	bu, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	client := Client{
		bu:      bu,
		funcMap: make(map[string]func(j JoinCompleted, data []byte) map[string]interface{}),
	}
	return &client, nil
}

func (c *Client) Do(ctx context.Context) error {
	if c.bu == nil {
		panic("base url is nil")
	}

	if c.bu.Scheme == "ws" || c.bu.Scheme == "wss" {
		return doConn(ctx, c)
	} else {
		return doReq(ctx, c)
	}
}

func doConn(ctx context.Context, c *Client) error {
	var conn *websocket.Conn
	{
		nc, err := newConn(c.bu)
		if err != nil {
			return err
		}
		conn = nc
	}

	for {
		select {
		case <-ctx.Done():
			return errors.New("done")

		default:
			_, data, err := conn.ReadMessage()
			if err != nil {
				return err
			}

			var j JoinCompleted
			err = json.Unmarshal(data, &j)
			if err != nil {
				return err
			}

			if funcCall, ok := c.funcMap[j.Msg]; ok {
				r := funcCall(j, data)
				if r != nil {
					marshal, _ := json.Marshal(r)
					err = conn.WriteMessage(websocket.TextMessage, marshal)
					if err != nil {
						return err
					}
				}
			}

			if funcCall, ok := c.funcMap["*"]; ok {
				r := funcCall(j, data)
				if r != nil {
					marshal, _ := json.Marshal(r)
					err = conn.WriteMessage(websocket.TextMessage, marshal)
					if err != nil {
						return err
					}
				}
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func doReq(ctx context.Context, c *Client) error {
	var response *http.Response
	{
		res, err := newReq(c.bu)
		if err != nil {
			return err
		}
		response = res
	}

	var data []byte
	before := []byte("data: ")
	reader := bufio.NewReader(response.Body)

	for {
		select {
		case <-ctx.Done():
			return errors.New("done")

		default:
			line, prefix, err := reader.ReadLine()
			if err != nil {
				return err
			}

			data = append(data, line...)
			if prefix {
				continue
			}

			if !bytes.HasPrefix(data, before) {
				data = nil
				continue
			}
			data = bytes.TrimPrefix(data, before)

			var j JoinCompleted
			err = json.Unmarshal(data, &j)
			if err != nil {
				return err
			}
			data = nil

			if funcCall, ok := c.funcMap[j.Msg]; ok {
				funcCall(j, data)
			}

			if funcCall, ok := c.funcMap["*"]; ok {
				funcCall(j, data)
			}

			if j.Success && j.Msg == "process_completed" {
				return nil
			}
		}
	}
}

func (c *Client) Event(eventId string, funcCall func(j JoinCompleted, data []byte) map[string]interface{}) {
	c.funcMap[eventId] = funcCall
}

func newReq(bu *url.URL) (*http.Response, error) {
	base := baseUrl(bu)
	response, err := http.DefaultClient.Get(base)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}

	return response, nil
}

func newConn(bu *url.URL) (*websocket.Conn, error) {
	base := baseUrl(bu)
	dialer := websocket.DefaultDialer
	conn, response, err := dialer.Dial(base, nil)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusSwitchingProtocols {
		return nil, errors.New(response.Status)
	}
	return conn, nil
}

func baseUrl(bu *url.URL) string {
	//h := ""
	//switch bu.Scheme {
	//case "http":
	//	h = "ws"
	//default:
	//	h = "wss"
	//}

	return fmt.Sprintf("%s://%s%s/queue/join?%s", bu.Scheme, bu.Host, bu.Path, bu.RawQuery)
}

func SessionHash() string {
	bin := "1234567890abcdefghijklmnopqrstuvwxyz"
	binL := len(bin)
	var buf []byte
	for x := 0; x < 10; x++ {
		ch := bin[rand.Intn(binL-1)]
		buf = append(buf, ch)
	}
	return string(buf)
}
