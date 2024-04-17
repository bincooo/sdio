package sdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"
)

// free SD: https://diffusionart.co
func init() {
	slog.SetDefault(slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	))
}

func TestJoin(t *testing.T) {
	base := "wss://krebzonide-sdxl-turbo-with-refiner.hf.space"
	client, err := New(base)
	if err != nil {
		t.Fatal(err)
	}

	timeout, withTimeout := context.WithTimeout(context.Background(), time.Minute)
	defer withTimeout()

	index := 0
	hash := SessionHash()
	value := ""

	client.Event("*", func(j JoinCompleted, data []byte) map[string]interface{} {
		t.Log(string(data))
		return nil
	})

	client.Event("send_hash", func(j JoinCompleted, data []byte) map[string]interface{} {
		return map[string]interface{}{
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	client.Event("send_data", func(j JoinCompleted, data []byte) map[string]interface{} {
		return map[string]interface{}{
			"data":         []interface{}{"1girl", 1, 3, -1, ""},
			"event_data":   nil,
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	client.Event("process_completed", func(j JoinCompleted, data []byte) map[string]interface{} {
		d := j.Output.Data
		if len(d) > 0 {
			inter, ok := d[0].([]interface{})
			if ok {
				result := inter[0].(map[string]interface{})
				if reflect.DeepEqual(result["is_file"], true) {
					value = result["name"].(string)
				}
			}
		}
		return nil
	})

	err = client.Do(timeout)
	if err != nil {
		t.Error(err)
	}

	t.Log(fmt.Sprintf("%s/file=%s", base, value))
}

func TestJoin2(t *testing.T) {
	base := "https://prodia-fast-stable-diffusion.hf.space"
	timeout, withTimeout := context.WithTimeout(context.Background(), time.Minute)
	defer withTimeout()

	index := 0
	hash := SessionHash()
	value := ""

	query := fmt.Sprintf("?fn_index=%d&session_hash=%s", index, hash)
	client, err := New(base + query)
	if err != nil {
		t.Fatal(err)
	}
	client.Event("*", func(j JoinCompleted, data []byte) map[string]interface{} {
		t.Log(string(data))
		return nil
	})

	client.Event("send_data", func(j JoinCompleted, data []byte) map[string]interface{} {
		obj := map[string]interface{}{
			"data": []interface{}{
				"space warrior, beautiful, female, ultrarealistic, soft lighting, 8k",
				"(deformed eyes, nose, ears, nose), bad anatomy, ugly",
				"anythingV5_PrtRE.safetensors [893e49b9]",
				20,
				"DPM++ 2M Karras",
				7,
				512,
				672,
				-1,
			},
			"event_data":   nil,
			"fn_index":     index,
			"session_hash": hash,
			"event_id":     j.EventId,
			"trigger_id":   rand.Intn(10) + 5,
		}
		marshal, _ := json.Marshal(obj)

		response, e := http.Post(base+"/queue/data", "application/json", bytes.NewReader(marshal))
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatal(response.Status)
		}
		return nil
	})

	client.Event("process_completed", func(j JoinCompleted, data []byte) map[string]interface{} {
		d := j.Output.Data
		if len(d) > 0 {
			result := d[0].(map[string]interface{})
			value = result["path"].(string)
		}
		return nil
	})

	err = client.Do(timeout)
	if err != nil {
		t.Error(err)
	}

	t.Log(fmt.Sprintf("%s/file=%s", base, value))
}

func TestHD(t *testing.T) {
	key := "xxx"
	url := "https://prodia-fast-stable-diffusion.hf.space/--replicas/kh9ul/file=/tmp/gradio/89b09665f21af786d3411a6707fdb3429741d766/18ba369e-4577-4ffb-b75e-1cf616b75566.png"
	jpg, err := HD(context.Background(), url, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(jpg)
}
