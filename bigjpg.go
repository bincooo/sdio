package sdio

import (
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/sdio/common"
	"net/http"
	"time"
)

const baseURL = "https://bigjpg.com/api/task/"

func Magnify(ctx context.Context, url, key string) (string, error) {
	payload := map[string]interface{}{
		"style": "art",
		"noise": "3",
		"x2":    "1",
		"input": url,
	}

	response, err := common.New().
		Context(ctx).
		URL(baseURL).
		Method(http.MethodPost).
		JsonHeader().
		Header("X-API-KEY", key).
		SetBody(payload).
		Do()
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.New("hd-api: " + response.Status)
	}

	if err = common.ToObj(response, &payload); err != nil {
		return "", err
	}

	var taskId string
	if tid, ok := payload["tid"]; ok {
		taskId = tid.(string)
	} else {
		if status, k := payload["status"]; k {
			return "", fmt.Errorf("hd-api: %s", status)
		}
		return "", errors.New("hd-api: fetch task failed")
	}

	retry := 20
	for {
		if retry < 0 {
			return "", errors.New("hd-api: poll failed")
		}
		retry--

		response, err = common.New().
			Context(ctx).
			URL(baseURL + taskId).
			Do()
		if err != nil {
			return "", err
		}

		if response.StatusCode != http.StatusOK {
			return "", errors.New("hd-api: " + response.Status)
		}

		if err = common.ToObj(response, &payload); err != nil {
			return "", err
		}

		if data, ok := payload[taskId]; ok {
			m := data.(map[string]interface{})
			if status, k := m["status"]; k {
				if status == "success" {
					return m["url"].(string), nil
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
}
