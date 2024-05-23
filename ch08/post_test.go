package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type User struct {
	First string
	Last  string
}

// POST 요청을 처리할 수 있는 함수를 반환
func handlePostUser(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func(r io.ReadCloser) {
			_, _ = io.Copy(io.Discard, r)
			_ = r.Close()
		}(r.Body)

		// 요청 메서드가 POST가 아닌 경우, 서버가 해당 메서드를 허용하지 않는다는 상태 코드를 반환
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}

		// 요청 body에 존재하는 JSON을 USER 객체로 디코딩 시도
		var u User
		err := json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			t.Error(err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// 디코딩 성공 시, 상태코드를 StatusAccepted로 설정
		w.WriteHeader(http.StatusAccepted)
	}
}

func TestPostUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlePostUser(t)))
	defer ts.Close()

	// 잘못된 타입의 요청(GET)을 전송
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	// 에러 확인
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d; actual status %d",
			http.StatusMethodNotAllowed, resp.StatusCode)
	}
	_ = resp.Body.Close()

	buf := new(bytes.Buffer)
	// User 객체를 JSON으로 인코딩한 후 바이트 버퍼에 쓰기
	u := User{First: "Adam", Last: "Woodbeck"}
	err = json.NewEncoder(buf).Encode(&u)
	if err != nil {
		t.Fatal(err)
	}
	// 바이트 버퍼에 포함된 요청 body의 데이터에 JSON이 포함되어 있음
	// Content-Type : application/json으로 설정
	// Content-Type : 서버의 핸들러가 요청 body로부터 데이터를 어떻게 처리할지 정의함
	resp, err = http.Post(ts.URL, "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	// 서버의 핸들러가 요청 body를 올바르게 디코딩하면, Accepted 상태코드를 얻음
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %d; actual status %d",
			http.StatusAccepted, resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMultipartPost(t *testing.T) {
	// 요청 body가 될 버퍼 생성
	reqBody := new(bytes.Buffer)
	// 버퍼를 래핑하는 멀티파트 writer를 생성
	w := multipart.NewWriter(reqBody)

	// 멀티파트 writer에 폼 필드 쓰기
	for k, v := range map[string]string{
		"date":        time.Now().Format(time.RFC3339),
		"description": "Form values with attached files",
	} {
		// 멀티파트 writer는 각 폼 필드를 해당하는 고유한 파트로 구분하고, 각 파트의 body로 바운더리와 헤더, 폼 필드의 값을 씀
		err := w.WriteField(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Attach files
	for i, file := range []string{
		"./files/hello.txt",
		"./files/goodbye.txt",
	} {
		filePart, err := w.CreateFormFile(fmt.Sprintf("file%d", i+1),
			filepath.Base(file))
		if err != nil {
			t.Fatal(err)
		}

		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.Copy(filePart, f)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}

	// 요청 body에 파트 추가가 완료되었으면, 반드시 멀티파트 writer를 닫아야 요청 body가 바운더리를 추가하는 작업을 올바르게 마무리함
	err := w.Close()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://httpbin.org/post", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d; actual status %d",
			http.StatusOK, resp.StatusCode)
	}

	t.Logf("\n%s", b)
}
