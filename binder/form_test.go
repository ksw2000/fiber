package binder

import (
	"bytes"
	"io"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func Test_FormBinder_Bind(t *testing.T) {
	t.Parallel()

	b := &FormBinding{
		EnableSplitting: true,
	}
	require.Equal(t, "form", b.Name())

	type Post struct {
		Title string `form:"title"`
	}

	type User struct {
		Name  string   `form:"name"`
		Names []string `form:"names"`
		Posts []Post   `form:"posts"`
		Age   int      `form:"age"`
	}
	var user User

	req := fasthttp.AcquireRequest()
	req.SetBodyString("name=john&names=john,doe&age=42&posts[0][title]=post1&posts[1][title]=post2&posts[2][title]=post3")
	req.Header.SetContentType("application/x-www-form-urlencoded")

	t.Cleanup(func() {
		fasthttp.ReleaseRequest(req)
	})

	err := b.Bind(req, &user)

	require.NoError(t, err)
	require.Equal(t, "john", user.Name)
	require.Equal(t, 42, user.Age)
	require.Len(t, user.Posts, 3)
	require.Equal(t, "post1", user.Posts[0].Title)
	require.Equal(t, "post2", user.Posts[1].Title)
	require.Equal(t, "post3", user.Posts[2].Title)
	require.Contains(t, user.Names, "john")
	require.Contains(t, user.Names, "doe")

	b.Reset()
	require.False(t, b.EnableSplitting)
}

func Benchmark_FormBinder_Bind(b *testing.B) {
	b.ReportAllocs()

	binder := &FormBinding{
		EnableSplitting: true,
	}

	type User struct {
		Name  string   `form:"name"`
		Posts []string `form:"posts"`
		Age   int      `form:"age"`
	}
	var user User

	req := fasthttp.AcquireRequest()
	req.SetBodyString("name=john&age=42&posts=post1,post2,post3")
	req.Header.SetContentType("application/x-www-form-urlencoded")

	var err error
	for b.Loop() {
		err = binder.Bind(req, &user)
	}

	require.NoError(b, err)
	require.Equal(b, "john", user.Name)
	require.Equal(b, 42, user.Age)
	require.Len(b, user.Posts, 3)
}

func Test_FormBinder_BindMultipart(t *testing.T) {
	t.Parallel()

	b := &FormBinding{
		EnableSplitting: true,
	}
	require.Equal(t, "form", b.Name())

	type Post struct {
		Title string `form:"title"`
	}

	type User struct {
		Avatar  *multipart.FileHeader   `form:"avatar"`
		Name    string                  `form:"name"`
		Names   []string                `form:"names"`
		Posts   []Post                  `form:"posts"`
		Avatars []*multipart.FileHeader `form:"avatars"`
		Age     int                     `form:"age"`
	}
	var user User

	req := fasthttp.AcquireRequest()

	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	require.NoError(t, mw.WriteField("name", "john"))
	require.NoError(t, mw.WriteField("names", "john,eric"))
	require.NoError(t, mw.WriteField("names", "doe"))
	require.NoError(t, mw.WriteField("age", "42"))
	require.NoError(t, mw.WriteField("posts[0][title]", "post1"))
	require.NoError(t, mw.WriteField("posts[1][title]", "post2"))
	require.NoError(t, mw.WriteField("posts[2][title]", "post3"))

	writer, err := mw.CreateFormFile("avatar", "avatar.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte("avatar"))
	require.NoError(t, err)

	writer, err = mw.CreateFormFile("avatars", "avatar1.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte("avatar1"))
	require.NoError(t, err)

	writer, err = mw.CreateFormFile("avatars", "avatar2.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte("avatar2"))
	require.NoError(t, err)

	require.NoError(t, mw.Close())

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())

	t.Cleanup(func() {
		fasthttp.ReleaseRequest(req)
	})

	err = b.Bind(req, &user)

	require.NoError(t, err)
	require.Equal(t, "john", user.Name)
	require.Equal(t, 42, user.Age)
	require.Contains(t, user.Names, "john")
	require.Contains(t, user.Names, "doe")
	require.Contains(t, user.Names, "eric")
	require.Len(t, user.Posts, 3)
	require.Equal(t, "post1", user.Posts[0].Title)
	require.Equal(t, "post2", user.Posts[1].Title)
	require.Equal(t, "post3", user.Posts[2].Title)

	require.NotNil(t, user.Avatar)
	require.Equal(t, "avatar.txt", user.Avatar.Filename)
	require.Equal(t, "application/octet-stream", user.Avatar.Header.Get("Content-Type"))

	file, err := user.Avatar.Open()
	require.NoError(t, err)

	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "avatar", string(content))

	require.Len(t, user.Avatars, 2)
	require.Equal(t, "avatar1.txt", user.Avatars[0].Filename)
	require.Equal(t, "application/octet-stream", user.Avatars[0].Header.Get("Content-Type"))

	file, err = user.Avatars[0].Open()
	require.NoError(t, err)

	content, err = io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "avatar1", string(content))

	require.Equal(t, "avatar2.txt", user.Avatars[1].Filename)
	require.Equal(t, "application/octet-stream", user.Avatars[1].Header.Get("Content-Type"))

	file, err = user.Avatars[1].Open()
	require.NoError(t, err)

	content, err = io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "avatar2", string(content))
}

func Benchmark_FormBinder_BindMultipart(b *testing.B) {
	b.ReportAllocs()

	binder := &FormBinding{
		EnableSplitting: true,
	}

	type User struct {
		Name  string   `form:"name"`
		Posts []string `form:"posts"`
		Age   int      `form:"age"`
	}
	var user User

	req := fasthttp.AcquireRequest()
	b.Cleanup(func() {
		fasthttp.ReleaseRequest(req)
	})

	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	require.NoError(b, mw.WriteField("name", "john"))
	require.NoError(b, mw.WriteField("age", "42"))
	require.NoError(b, mw.WriteField("posts", "post1"))
	require.NoError(b, mw.WriteField("posts", "post2"))
	require.NoError(b, mw.WriteField("posts", "post3"))
	require.NoError(b, mw.Close())

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())

	var err error
	for b.Loop() {
		err = binder.Bind(req, &user)
	}

	require.NoError(b, err)
	require.Equal(b, "john", user.Name)
	require.Equal(b, 42, user.Age)
	require.Len(b, user.Posts, 3)
}
