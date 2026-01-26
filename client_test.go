package http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_Post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		fmt.Fprintln(w, string(b))
	}))
	defer ts.Close()

	cli := NewClient(nil)

	in := struct {
		Name string `json:"name"`
	}{
		Name: "Name",
	}
	out := struct {
		Name string `json:"name"`
	}{}
	err := cli.Post(ts.URL, in, &out)
	assert.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestClient_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name":"Name"}`)
	}))
	defer ts.Close()

	cli := NewClient(nil)

	t.Run("out is struct", func(t *testing.T) {
		user := struct {
			Name string `json:"name"`
		}{}
		err := cli.Get(ts.URL, &user)
		assert.NoError(t, err)
		assert.Equal(t, "Name", user.Name)
	})
	t.Run("out is Writer", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := cli.Get(ts.URL, buf)
		assert.NoError(t, err)
		assert.Equal(t, `{"name":"Name"}`+"\n", buf.String())
	})
}

func TestClient_Do_Status200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name":"Name"}`)
	}))
	defer ts.Close()

	client := NewClient(nil)

	t.Run("nil", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.URL, nil)
		assert.NoError(t, err)

		_, err = client.Do(context.Background(), req, nil)
		assert.Nil(t, err)
	})
	t.Run("Writer", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.URL, nil)
		assert.NoError(t, err)

		buf := new(bytes.Buffer)
		_, err = client.Do(context.Background(), req, buf)
		assert.Nil(t, err)
		assert.Equal(t, `{"name":"Name"}`+"\n", buf.String())
	})
	t.Run("struct", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.URL, nil)
		assert.NoError(t, err)

		v := &struct {
			Name string `json:"name"`
		}{}
		_, err = client.Do(context.Background(), req, v)
		assert.Nil(t, err)
		assert.Equal(t, "Name", v.Name)
	})
}
func TestClient_Do_Status400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	client := NewClient(nil)

	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err)
	_, err = client.Do(context.Background(), req, nil)
	assert.NotNil(t, err)
}

func TestCheckResponse(t *testing.T) {
	t.Run("status 200", func(t *testing.T) {
		res := &http.Response{
			StatusCode: 200,
		}
		assert.NoError(t, CheckResponse(res))
	})

	t.Run("status 400", func(t *testing.T) {
		res := &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(bytes.NewBufferString("test msg")),
		}
		err := CheckResponse(res)
		assert.NotNil(t, err)
		errRes, ok := err.(*ErrorResponse)
		assert.Truef(t, ok, "%q is not *ErrorResponse", err)
		assert.Equal(t, "test msg", errRes.Message)
	})
}

func TestClient_NewRequest_POST(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"
	user := struct {
		Name string `json:"name"`
	}{
		Name: "testName",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
		assert.Equal(t, "application/json", r.Header["Content-Type"][0])

		v := struct {
			Name string `json:"name"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&v)
		assert.NoError(t, err)
		assert.Equal(t, user, v)
	}))
	defer ts.Close()

	client, err := New(nil,
		WithBaseURL(ts.URL),
		WithUserAgent(userAgent),
		WithAuthorization(token))
	assert.NoError(t, err)

	req, err := client.NewRequest("POST", "user", user)
	assert.NoError(t, err)
	assert.Equal(t, ts.URL+"/user", req.URL.String())
	assert.Equal(t, req.Method, "POST")
	assert.NotNil(t, req.Body)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	test_client(t, client)
}

func TestClient_NewRequest_WithoutBaseURL(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
	}))
	defer ts.Close()

	client, err := New(nil,
		WithUserAgent(userAgent),
		WithAuthorization(token))
	assert.NoError(t, err)

	req, err := client.NewRequest("GET", ts.URL+"/user", nil)
	assert.NoError(t, err)
	assert.Equal(t, ts.URL+"/user", req.URL.String())
	assert.Equal(t, req.Method, "GET")
	assert.Nil(t, req.Body)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	test_client(t, client)
}

func TestClient_NewRequest_GET(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
	}))
	defer ts.Close()

	client, err := New(nil,
		WithBaseURL(ts.URL),
		WithUserAgent(userAgent),
		WithAuthorization(token))
	assert.NoError(t, err)

	req, err := client.NewRequest("GET", "user", nil)
	assert.NoError(t, err)
	assert.Equal(t, ts.URL+"/user", req.URL.String())
	assert.Equal(t, req.Method, "GET")
	assert.Nil(t, req.Body)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	test_client(t, client)
}

func Test_New(t *testing.T) {
	client, err := New(nil,
		WithBaseURL("https://golang.org/"),
		WithUserAgent("custome"),
		WithAuthorization("token bG9sOnNlY3VyZQ"),
		WithInterceptor(DefaultInterceptor))

	assert.NoError(t, err)
	assert.Equal(t, "golang.org", client.BaseURL.Host)
	assert.Truef(t, len(client.client.Transport.(*interTransport).interceptors) == 1,
		"len=%d", len(client.client.Transport.(*interTransport).interceptors))

	test_client(t, client)
}

func Test_NewClient(t *testing.T) {
	t.Run("httpClient is nil", func(t *testing.T) {
		client := NewClient(nil)

		test_client(t, client)

		assert.NotNil(t, client.client.Transport)

		tr, ok := client.client.Transport.(*interTransport)
		assert.Truef(t, ok, "Transport is not interTransport")

		assert.Equal(t, http.DefaultTransport, tr.transport)
	})
	t.Run("custom client and transport", func(t *testing.T) {
		custTr := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 20 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost: 100,
		}
		custClient := &http.Client{
			Transport: custTr,
			Timeout:   5 * time.Second,
		}

		client := NewClient(custClient)

		test_client(t, client)

		assert.Equal(t, custClient, client.client)
		assert.NotNil(t, client.client.Transport)

		tr, ok := client.client.Transport.(*interTransport)
		assert.Truef(t, ok, "Transport is not interTransport")

		assert.Equal(t, custTr, tr.transport)
	})
}

func TestClient_AddInterceptor(t *testing.T) {
	client := NewClient(nil)

	var got string
	err := client.AddInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		got = "AddInterceptor"
		return handler(req)
	})
	assert.NoError(t, err)

	test_client(t, client)

	assert.Equal(t, "AddInterceptor", got)
}

func test_client(t *testing.T, client *Client) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name":"Name"}`)
	}))
	defer ts.Close()

	res, err := client.client.Get(ts.URL)
	assert.NoError(t, err)

	got, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	res.Body.Close()

	want := `{"name":"Name"}` + "\n"
	assert.Equal(t, want, string(got))
}

func TestClient_PostMultipart(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		assert.NoError(t, err)

		// Check fields
		description := r.MultipartForm.Value["description"]
		assert.Equal(t, []string{"my file"}, description)

		// Check files
		fileHeaders := r.MultipartForm.File["attachment"]
		assert.Len(t, fileHeaders, 1)

		file, err := fileHeaders[0].Open()
		assert.NoError(t, err)
		defer file.Close()

		content, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success":true}`)
	}))
	defer ts.Close()

	cli := NewClient(nil)

	t.Run("single file with fields", func(t *testing.T) {
		form := NewMultipartForm()
		form.AddField("description", "my file")
		form.AddFile("attachment", "photo.jpg", strings.NewReader("file content"))

		var resp struct {
			Success bool `json:"success"`
		}
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("only fields", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			assert.NoError(t, err)

			field1 := r.MultipartForm.Value["field1"]
			field2 := r.MultipartForm.Value["field2"]
			assert.Equal(t, []string{"value1"}, field1)
			assert.Equal(t, []string{"value2"}, field2)

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"ok":true}`)
		}))
		defer ts.Close()

		form := NewMultipartForm()
		form.AddField("field1", "value1")
		form.AddField("field2", "value2")

		var resp struct {
			OK bool `json:"ok"`
		}
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
	})

	t.Run("multiple files same field", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			assert.NoError(t, err)

			fileHeaders := r.MultipartForm.File["files"]
			assert.Len(t, fileHeaders, 2)

			// Check first file
			file1, err := fileHeaders[0].Open()
			assert.NoError(t, err)
			content1, err := io.ReadAll(file1)
			file1.Close()
			assert.NoError(t, err)
			assert.Equal(t, "content1", string(content1))

			// Check second file
			file2, err := fileHeaders[1].Open()
			assert.NoError(t, err)
			content2, err := io.ReadAll(file2)
			file2.Close()
			assert.NoError(t, err)
			assert.Equal(t, "content2", string(content2))

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"count":2}`)
		}))
		defer ts.Close()

		form := NewMultipartForm()
		form.AddFile("files", "file1.txt", strings.NewReader("content1"))
		form.AddFile("files", "file2.txt", strings.NewReader("content2"))

		var resp struct {
			Count int `json:"count"`
		}
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.Count)
	})

	t.Run("only files", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			assert.NoError(t, err)

			fileHeaders := r.MultipartForm.File["file"]
			assert.Len(t, fileHeaders, 1)

			file, err := fileHeaders[0].Open()
			assert.NoError(t, err)
			content, err := io.ReadAll(file)
			file.Close()
			assert.NoError(t, err)
			assert.Equal(t, "file only", string(content))

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"empty":false}`)
		}))
		defer ts.Close()

		form := NewMultipartForm()
		form.AddFile("file", "data.bin", strings.NewReader("file only"))

		var resp struct {
			Empty bool `json:"empty"`
		}
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Empty)
	})

	t.Run("boundary is unique per request", func(t *testing.T) {
		// Test that each multipart request gets a unique boundary
		form1 := NewMultipartForm()
		form1.AddField("test", "value1")
		body1, ct1, err1 := form1.buildMultipartBody()
		assert.NoError(t, err1)
		assert.NotNil(t, body1)
		assert.Contains(t, ct1, "multipart/form-data; boundary=")

		form2 := NewMultipartForm()
		form2.AddField("test", "value2")
		body2, ct2, err2 := form2.buildMultipartBody()
		assert.NoError(t, err2)
		assert.NotNil(t, body2)
		assert.Contains(t, ct2, "multipart/form-data; boundary=")

		// Boundaries should be different
		assert.NotEqual(t, ct1, ct2, "multipart boundaries should be unique per request")
	})
}

func TestClient_NewMultipartRequest(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary="))

		err := r.ParseMultipartForm(10 << 20)
		assert.NoError(t, err)

		field := r.MultipartForm.Value["test"]
		assert.Equal(t, []string{"value"}, field)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true}`)
	}))
	defer ts.Close()

	client, err := New(nil,
		WithBaseURL(ts.URL),
		WithUserAgent(userAgent),
		WithAuthorization(token))
	assert.NoError(t, err)

	form := NewMultipartForm()
	form.AddField("test", "value")

	req, err := client.NewMultipartRequest("POST", "upload", form)
	assert.NoError(t, err)
	assert.Equal(t, ts.URL+"/upload", req.URL.String())
	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, req.Body)
	contentType := req.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(contentType, "multipart/form-data; boundary="))
	assert.Contains(t, contentType, "boundary=")

	var resp struct {
		OK bool `json:"ok"`
	}
	_, err = client.Do(context.Background(), req, &resp)
	assert.NoError(t, err)
	assert.True(t, resp.OK)

	test_client(t, client)
}

func TestMultipartForm_EdgeCases(t *testing.T) {
	t.Run("empty form", func(t *testing.T) {
		form := NewMultipartForm()
		body, contentType, err := form.buildMultipartBody()
		assert.NoError(t, err)
		assert.NotNil(t, body)
		assert.Contains(t, contentType, "multipart/form-data; boundary=")

		// Read the body to verify it's not empty
		content, err := io.ReadAll(body)
		assert.NoError(t, err)
		assert.True(t, len(content) > 0, "empty form should still create valid multipart body")
	})

	t.Run("multiple fields same name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			assert.NoError(t, err)

			values := r.MultipartForm.Value["tags"]
			assert.Len(t, values, 3)
			assert.Contains(t, values, "go")
			assert.Contains(t, values, "http")
			assert.Contains(t, values, "multipart")

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"count":3}`)
		}))
		defer ts.Close()

		form := NewMultipartForm()
		form.AddField("tags", "go")
		form.AddField("tags", "http")
		form.AddField("tags", "multipart")

		var resp struct {
			Count int `json:"count"`
		}
		cli := NewClient(nil)
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 3, resp.Count)
	})

	t.Run("large number of fields", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			assert.NoError(t, err)

			// Check all 10 fields
			for i := 1; i <= 10; i++ {
				fieldName := fmt.Sprintf("field%d", i)
				values := r.MultipartForm.Value[fieldName]
				assert.Len(t, values, 1)
				assert.Equal(t, fmt.Sprintf("value%d", i), values[0])
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"ok":true}`)
		}))
		defer ts.Close()

		form := NewMultipartForm()
		for i := 1; i <= 10; i++ {
			form.AddField(fmt.Sprintf("field%d", i), fmt.Sprintf("value%d", i))
		}

		var resp struct {
			OK bool `json:"ok"`
		}
		cli := NewClient(nil)
		err := cli.PostMultipart(ts.URL, form, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
	})
}
