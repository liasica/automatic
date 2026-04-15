// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package service

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"automatic/dashboard"
	"automatic/service/handler"
)

// Server HTTP 服务器
type Server struct {
	echo *echo.Echo
	addr string
}

// NewServer 创建 HTTP 服务器
func NewServer(addr string, attendance *handler.Attendance) *Server {
	e := echo.New()

	// 隐藏 Banner 和端口输出
	e.HideBanner = true
	e.HidePort = true

	// 中间件
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType},
	}))

	// API 路由
	api := e.Group("/api")
	api.GET("/users", attendance.Users)
	records := api.Group("/attendance/records")
	records.GET("", attendance.Query)
	records.POST("", attendance.Create)
	records.DELETE("", attendance.Delete)

	// 嵌入前端静态文件，子目录 dist 作为根
	distFS, _ := fs.Sub(dashboard.Dist, "dist")
	staticHandler := http.FileServer(http.FS(distFS))

	// 所有非 API 请求回退到前端静态文件（SPA 路由）
	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 尝试直接提供静态文件
		f, err := distFS.Open(r.URL.Path[1:]) // 去掉前导 /
		if err != nil {
			// 文件不存在，回退到 index.html（SPA 路由）
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		staticHandler.ServeHTTP(w, r)
	})))

	return &Server{
		echo: e,
		addr: addr,
	}
}

// Start 启动 HTTP 服务器
func (s *Server) Start() error {
	return s.echo.Start(s.addr)
}

// Shutdown 优雅关闭 HTTP 服务器
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}
