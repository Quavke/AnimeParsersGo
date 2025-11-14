package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	errs "github.com/Quavke/AnimeParsersGo/errors"
	"github.com/Quavke/AnimeParsersGo/models"
)

type RequestResult struct {
	Data     []byte
	Json     models.JSONResponse
	Response *http.Response
}

// Функция для выполнения HTTP-запросов с контекстом. Any - string или models.JSONResponse
func RequestWithContext(context context.Context, method, URL string, params models.Params, headers models.Headers, jsonResp bool, jsonType models.JSONResponse) (*RequestResult, error) {
	url_params := url.Values{}

	if params != nil {
		for key, value := range params {
			url_params.Set(key, value)
		}
		URL = URL + "?" + url_params.Encode()
	}

	request, err := http.NewRequestWithContext(context, method, URL, nil)

	if err != nil {
		error_message := fmt.Sprintf("Request error : не смог создать request. Ошибка: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
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
		return nil, errs.NewServiceError(error_message)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errs.NewTooManyRequestsError("Request error : GetTranslationsInfo : Сервер вернул код ошибки 429. Слишком частые запросы")
	}
	if resp.StatusCode == 520 {
		return nil, errs.NewServiceIsOverloadedError("Request error : GetTranslationsInfo : Сервер вернул статус ответа 520, что означает что он перегружен и не может ответить сразу")
	}
	if resp.StatusCode != http.StatusOK {
		error_message := fmt.Sprintf("Request error : Сервер не вернул ожидаемый код 200. Код: %d", resp.StatusCode)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

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
