package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	errs "github.com/Quavke/AnimeParsersGo/errors"
	"github.com/Quavke/AnimeParsersGo/models"
)

type RequestResult struct {
	Data     []byte
	Json     models.JSONResponse
	Response *http.Response
}

var (
	numWorkers  = 3
	maxAttempts = 10
)

type worker_params struct {
	ctx     context.Context
	method  string
	URL     string
	params  models.Params
	headers models.Headers
}

func worker(w_params *worker_params, ch chan<- *http.Response, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	url_params := url.Values{}

	if w_params.params != nil {
		for key, value := range w_params.params {
			url_params.Set(key, value)
		}
		w_params.URL = w_params.URL + "?" + url_params.Encode()
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	request, err := http.NewRequestWithContext(w_params.ctx, w_params.method, w_params.URL, nil)

	if err != nil {
		error_message := fmt.Sprintf("Request error : %d : http не смог создать request. Ошибка: %v", id, err)
		log.Println(error_message)
		return
	}

	for key, value := range w_params.headers {
		request.Header.Set(key, value)
	}

	var resp *http.Response
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-w_params.ctx.Done():
			return
		default:
		}

		resp, err = client.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Request error : %d : http клиент не смог выполнить запрос. Попытка %d", id, attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Request error : %d : http клиент не смог выполнить запрос. Попытка %d", id, attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else if resp.StatusCode == http.StatusTooManyRequests {
			error_message := fmt.Sprintf("Request error : %d : клиент получил ответ со статусом StatusTooManyRequests. Попытка %d", id, attempt)
			log.Println(error_message)
			return
		} else if resp.StatusCode == 520 {
			error_message := fmt.Sprintf("Request error : %d : клиент получил ответ со статусом 520 (Cloudflare не смог обработать ответ от исходного веб-сервера). Попытка %d", id, attempt)
			log.Println(error_message)
			return
		} else {
			break
		}
	}

	if err != nil {
		error_message := fmt.Sprintf("Request error : %d : http клиент не смог выполнить запрос. Ошибка: %v", id, err)
		log.Println(error_message)
		return
	}

	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Request error : %d : Сервер не вернул ожидаемый код 200. Код: %d", id, resp.StatusCode)
		log.Println(error_message)
		return
	}

	select {
	case ch <- resp:
		return
	default:
		return
	}
}

func RequestWithContext(ctx context.Context, method, URL string, params models.Params, headers models.Headers, jsonResp bool, jsonType models.JSONResponse) (*RequestResult, error) {
	result := make(chan *http.Response, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	w_params := &worker_params{
		ctx:     ctx,
		method:  method,
		URL:     URL,
		params:  params,
		headers: headers,
	}

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(w_params, result, wg, i)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	if resp, ok := <-result; ok {
		req_result := &RequestResult{Response: resp}

		if jsonResp {
			if err := jsonType.Decode(resp.Body); err != nil {
				return nil, errs.NewJsonDecodeFailureError(fmt.Sprintf("Request error : ошибка декодирования json: %v", err))
			}
			req_result.Json = jsonType
			return req_result, nil
		} else {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				error_message := fmt.Sprintf("Request error : не удалось прочитать тело ответа. Ошибка: %v", err)
				log.Println(error_message)
				return nil, errs.NewServiceError(error_message)
			}
			req_result.Data = bodyBytes
			return req_result, nil
		}
	} else {
		error_message := "Request error : ни один воркер не вернул ответ"
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

}

func TestURL(URL, method string, params models.Params, headers models.Headers) error {
	url_params := url.Values{}

	if params != nil {
		for key, value := range params {
			url_params.Set(key, value)
		}
		URL = URL + "?" + url_params.Encode()
	}

	request, err := http.NewRequest(method, URL, nil)

	if err != nil {
		error_message := fmt.Sprintf("Request error : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return errs.NewServiceError(error_message)
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	var resp *http.Response
	for attempt := 1; attempt <= 50; attempt++ {
		resp, err = http.DefaultClient.Do(request)
		if err != nil {
			error_message := fmt.Sprintf("Request error : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			continue
		} else if resp.StatusCode != http.StatusOK {
			error_message := fmt.Sprintf("Request error : http клиент не смог выполнить запрос. Попытка %d", attempt)
			log.Println(error_message)
			resp.Body.Close()
			continue
		} else {
			break
		}
	}

	if err != nil {
		error_message := fmt.Sprintf("Request error : http клиент не смог выполнить запрос. Ошибка: %v", err)
		log.Println(error_message)
		return errs.NewServiceError(error_message)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return errs.NewTooManyRequestsError("Request error : GetTranslationsInfo : Сервер вернул код ошибки 429. Слишком частые запросы")
	}
	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Request error : Сервер не вернул ожидаемый код 200. Код: %d", resp.StatusCode)
		log.Println(error_message)
		return errs.NewServiceError(error_message)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		error_message := fmt.Sprintf("Request error : не удалось прочитать тело ответа. Ошибка: %v", err)
		log.Println(error_message)
		return errs.NewServiceError(error_message)
	}

	fmt.Println(string(bodyBytes))
	return nil
}
