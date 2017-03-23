// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	router "aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

type (
	Anonymous1 struct {
		Name string
	}

	Func1 func(e *Event)

	Level1 struct{ *Context }

	Level2 struct{ Level1 }

	Level3 struct{ Level2 }

	Level4 struct{ Level3 }

	Path1 struct {
		Anonymous Anonymous1
		*Context
	}

	Path2 struct {
		Level1
		Path1
		Level4
		Func1
	}
)

func TestContextReverseURL(t *testing.T) {
	appCfg, _ := config.ParseString("")
	err := initRoutes(getTestdataPath(), appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	ctx := &Context{
		Req: getAahRequest("localhost:8080", "GET", "/doc/v0.3/mydoc.html", ""),
	}

	reverseURL := ctx.ReverseURL("version_home", "v0.1")
	assert.Equal(t, "//localhost:8080/doc/v0.1", reverseURL)

	reverseURL = ctx.ReverseURLm("show_doc", map[string]interface{}{
		"version": "v0.2",
		"content": "getting-started.html",
	})
	assert.Equal(t, "//localhost:8080/doc/v0.2/getting-started.html", reverseURL)

	ctx.Reset()
}

func TestContextViewArgs(t *testing.T) {
	ctx := &Context{viewArgs: make(map[string]interface{}, 0)}

	ctx.AddViewArg("key1", "key1 value")
	assert.Equal(t, "key1 value", ctx.viewArgs["key1"])
	assert.Nil(t, ctx.viewArgs["notexists"])
}

func TestContextMsg(t *testing.T) {
	err := initI18n(getTestdataPath())
	assert.Nil(t, err)
	assert.NotNil(t, AppI18n())

	ctx := &Context{
		Req: getAahRequest("localhost:8080", "GET", "/doc/v0.3/mydoc.html", "en-us;q=0.0,en;q=0.7, da, en-gb;q=0.8"),
	}

	msg := ctx.Msg("label.pages.site.get_involved.title")
	assert.Equal(t, "", msg)

	msg = ctx.Msgl(ahttp.ToLocale(&ahttp.AcceptSpec{Value: "en-US", Raw: "en-US"}), "label.pages.site.get_involved.title")
	assert.Equal(t, "Get Involved - aah web framework for Go", msg)

	ctx.Req = getAahRequest("localhost:8080", "GET", "/doc/v0.3/mydoc.html", "en-us;q=0.0,en;q=0.7,en-gb;q=0.8")
	msg = ctx.Msg("label.pages.site.get_involved.title")
	assert.Equal(t, "Get Involved - aah web framework for Go", msg)

	ctx.Reset()
}

func TestContextSetTarget(t *testing.T) {
	addToCRegistry()

	ctx := &Context{}

	err1 := ctx.setTarget(&router.Route{Controller: "Level3", Action: "Testing"})
	assert.Nil(t, err1)
	assert.Equal(t, "Level3", ctx.controller)
	assert.NotNil(t, ctx.action)
	assert.Equal(t, "Testing", ctx.action.Name)
	assert.NotNil(t, ctx.action.Parameters)
	assert.Equal(t, "userId", ctx.action.Parameters[0].Name)

	err2 := ctx.setTarget(&router.Route{Controller: "NoController"})
	assert.Equal(t, errTargetNotFound, err2)

	err3 := ctx.setTarget(&router.Route{Controller: "Level3", Action: "NoAction"})
	assert.Equal(t, errTargetNotFound, err3)
}

func TestContextAbort(t *testing.T) {
	ctx := &Context{}

	assert.False(t, ctx.abort)
	ctx.Abort()
	assert.True(t, ctx.abort)
}

func TestContextNil(t *testing.T) {
	ctx := &Context{}

	assert.Nil(t, ctx.Reply())
	assert.Nil(t, ctx.ViewArgs())
}

func TestContextEmbeddedAndController(t *testing.T) {
	addToCRegistry()

	assertEmbeddedIndexes(t, Level1{}, [][]int{{0}})
	assertEmbeddedIndexes(t, Level2{}, [][]int{{0, 0}})
	assertEmbeddedIndexes(t, Level3{}, [][]int{{0, 0, 0}})
	assertEmbeddedIndexes(t, Level4{}, [][]int{{0, 0, 0, 0}})
	assertEmbeddedIndexes(t, Path1{}, [][]int{{1}})
	assertEmbeddedIndexes(t, Path2{}, [][]int{{0, 0}, {1, 1}, {2, 0, 0, 0, 0}})
}

func assertEmbeddedIndexes(t *testing.T, c interface{}, expected [][]int) {
	actual := findEmbeddedContext(reflect.TypeOf(c))
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match. expected %v actual %v", expected, actual)
	}
}

func addToCRegistry() {
	cRegistry = controllerRegistry{}

	AddController((*Level1)(nil), []*MethodInfo{
		&MethodInfo{
			Name:       "Index",
			Parameters: []*ParameterInfo{},
		},
	})
	AddController((*Level2)(nil), []*MethodInfo{
		&MethodInfo{
			Name:       "Scope",
			Parameters: []*ParameterInfo{},
		},
	})
	AddController((*Level3)(nil), []*MethodInfo{
		&MethodInfo{
			Name: "Testing",
			Parameters: []*ParameterInfo{
				&ParameterInfo{
					Name: "userId",
					Type: reflect.TypeOf((*int)(nil)),
				},
			},
		},
	})
	AddController((*Level4)(nil), nil)
	AddController((*Path1)(nil), nil)
	AddController((*Path2)(nil), nil)
}

func getAahRequest(host, method, urlStr, al string) *ahttp.Request {
	reqURL, _ := url.Parse(urlStr)

	rawReq := &http.Request{
		Host:   host,
		URL:    reqURL,
		Method: method,
		Header: http.Header{},
	}
	rawReq.Header.Add(ahttp.HeaderAcceptLanguage, al)

	return ahttp.ParseRequest(rawReq, &ahttp.Request{})
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
