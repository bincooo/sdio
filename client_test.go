package sdio

import (
	"context"
	"fmt"
	"log/slog"
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
	base := "https://krebzonide-sdxl-turbo-with-refiner.hf.space"
	client, err := New(base)
	if err != nil {
		t.Fatal(err)
	}

	timeout, withTimeout := context.WithTimeout(context.Background(), time.Minute)
	defer withTimeout()

	index := 0
	hash := SessionHash()
	value := ""

	client.Event("*", func(j joinCompleted, data []byte) map[string]interface{} {
		t.Log(string(data))
		return nil
	})

	client.Event("send_hash", func(j joinCompleted, data []byte) map[string]interface{} {
		return map[string]interface{}{
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	client.Event("send_data", func(j joinCompleted, data []byte) map[string]interface{} {
		return map[string]interface{}{
			"data":         []interface{}{"1girl", 1, 3, -1, ""},
			"event_data":   nil,
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	client.Event("process_completed", func(j joinCompleted, data []byte) map[string]interface{} {
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
