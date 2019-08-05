package client

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSetDefaultHeader(t *testing.T) {
	type args struct {
		header http.Header
		key    string
		value  string
	}
	tests := []struct {
		name string
		args args
		want http.Header
	}{
		{
			name: "Simple",
			args: args{http.Header{}, "key", "value"},
			want: http.Header{"Key": []string{"value"}},
		},
		{
			name: "Dont Override",
			args: args{http.Header{"Key": []string{"value"}}, "key", "new value"},
			want: http.Header{"Key": []string{"value"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetDefaultHeader(tt.args.header, tt.args.key, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetDefaultHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertHeaderToMap(t *testing.T) {
	type args struct {
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "Simple",
			args: args{http.Header{"Key": []string{"value"}}},
			want: map[string]interface{}{"Key": "value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertHeaderToMap(tt.args.header); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertHeaderToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertMapToHeader(t *testing.T) {
	type args struct {
		m map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want http.Header
	}{
		{
			name: "Simple",
			args: args{map[string]interface{}{"Key": "value"}},
			want: http.Header{"Key": []string{"value"}},
		},
		{
			name: "Non standard key",
			args: args{map[string]interface{}{"key": "value"}},
			want: http.Header{"Key": []string{"value"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertMapToHeader(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertMapToHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCharsetFromHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil)
	res, _ := NewClient(DefaultMaxBody, false, DefaultRetryTimes, DefaultRetryHTTPCodes, "").DoRequest(req)

	if string(res.Body) != "Gültekin" {
		t.Fatal(string(res.Body))
	}
}

func TestCharsetFromBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil)
	res, _ := NewClient(DefaultMaxBody, false, DefaultRetryTimes, DefaultRetryHTTPCodes, "").DoRequest(req)

	if string(res.Body) != "Gültekin" {
		t.Fatal(string(res.Body))
	}
}

func TestCharsetProvidedWithRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil)
	req.Encoding = "windows-1254"
	res, _ := NewClient(DefaultMaxBody, false, DefaultRetryTimes, DefaultRetryHTTPCodes, "").DoRequest(req)

	if string(res.Body) != "Gültekin" {
		t.Fatal(string(res.Body))
	}
}

func TestRetry(t *testing.T) {
	req, _ := NewRequest("GET", "https://httpbin.org/status/500", nil)
	res, err := NewClient(DefaultMaxBody, false, DefaultRetryTimes, DefaultRetryHTTPCodes, "").DoRequest(req)
	assert.Nil(t, res)
	assert.Error(t, err)
}
