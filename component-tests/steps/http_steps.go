package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cucumber/godog"
)

// registerHTTPSteps регистрирует универсальные HTTP-степы:
// отправка запроса (с/без тела, с/без авторизации) и проверки ответа
// (статус, заголовки, JSON-поля).
func (w *World) registerHTTPSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^клиент отправляет (GET|POST|PUT|DELETE) (\S+)$`, w.sendRequest)
	ctx.Step(`^клиент отправляет (GET|POST|PUT|DELETE) (\S+) с телом:$`, w.sendRequestWithBody)
	ctx.Step(`^ответ (\d+)(?:\s|$)`, w.responseStatus)
	ctx.Step(`^ответ содержит заголовок ([A-Za-z\-]+)$`, w.responseHasHeader)
	ctx.Step(`^ответ содержит JSON-поле ([\w\.\[\]]+) со значением "([^"]*)"$`, w.responseJSONField)
	ctx.Step(`^ответ содержит непустое JSON-поле ([\w\.\[\]]+)$`, w.responseJSONFieldNonEmpty)
}

// sendRequest выполняет HTTP-запрос без тела. Если в World уже есть
// accessToken — добавляет заголовок Authorization: Bearer.
func (w *World) sendRequest(method, path string) error {
	return w.doRequest(method, path, nil)
}

// sendRequestWithBody — то же, но с DocString-телом из шага.
func (w *World) sendRequestWithBody(method, path string, body *godog.DocString) error {
	return w.doRequest(method, path, []byte(body.Content))
}

func (w *World) doRequest(method, path string, body []byte) error {
	url := w.serviceBaseURL + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if w.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+w.accessToken)
	}

	// Закрываем предыдущий response, чтобы не текли коннекты.
	if w.lastResponse != nil {
		_ = w.lastResponse.Body.Close()
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send %s %s: %w", method, url, err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("read body: %w", err)
	}

	w.lastResponse = resp
	w.lastBody = respBody
	return nil
}

// responseStatus проверяет HTTP-код последнего ответа.
func (w *World) responseStatus(expected int) error {
	if w.lastResponse == nil {
		return fmt.Errorf("ответ ещё не получен")
	}
	if w.lastResponse.StatusCode != expected {
		return fmt.Errorf("ожидали статус %d, получили %d (тело: %s)",
			expected, w.lastResponse.StatusCode, string(w.lastBody))
	}
	return nil
}

// responseHasHeader проверяет, что в ответе присутствует заголовок (значение не сверяется).
func (w *World) responseHasHeader(name string) error {
	if w.lastResponse == nil {
		return fmt.Errorf("ответ ещё не получен")
	}
	if w.lastResponse.Header.Get(name) == "" {
		return fmt.Errorf("ожидали заголовок %q, не нашли в ответе", name)
	}
	return nil
}

// responseJSONFieldNonEmpty проверяет, что JSON-поле ответа присутствует и непустое.
func (w *World) responseJSONFieldNonEmpty(field string) error {
	if w.lastBody == nil {
		return fmt.Errorf("ответ ещё не получен")
	}
	var parsed map[string]any
	if err := json.Unmarshal(w.lastBody, &parsed); err != nil {
		return fmt.Errorf("ответ не валидный JSON: %w (тело: %s)", err, string(w.lastBody))
	}
	v, ok := parsed[field]
	if !ok {
		return fmt.Errorf("в ответе нет поля %q (тело: %s)", field, string(w.lastBody))
	}
	if fmt.Sprintf("%v", v) == "" {
		return fmt.Errorf("поле %q пустое", field)
	}
	return nil
}

// responseJSONField проверяет, что JSON-поле ответа имеет ожидаемое значение.
// Поддерживает плоские пути (без вложенных объектов).
func (w *World) responseJSONField(field, expected string) error {
	if w.lastBody == nil {
		return fmt.Errorf("ответ ещё не получен")
	}
	var parsed map[string]any
	if err := json.Unmarshal(w.lastBody, &parsed); err != nil {
		return fmt.Errorf("ответ не валидный JSON: %w (тело: %s)", err, string(w.lastBody))
	}
	v, ok := parsed[field]
	if !ok {
		return fmt.Errorf("в ответе нет поля %q (тело: %s)", field, string(w.lastBody))
	}
	got := fmt.Sprintf("%v", v)
	if got != expected {
		return fmt.Errorf("поле %q: ожидали %q, получили %q", field, expected, got)
	}
	return nil
}
