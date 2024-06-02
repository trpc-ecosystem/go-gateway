package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/stream", streamHandler)
	http.ListenAndServe(":8081", nil)
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头，指定内容类型为text/event-stream
	w.Header().Set("Content-Type", "text/event-stream")
	// 设置响应头，禁用HTTP缓存
	w.Header().Set("Cache-Control", "no-cache")
	// 设置响应头，允许跨域访问
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建一个新的事件流
	eventStream := make(chan string)

	// 启动一个goroutine，用于向客户端发送事件
	go func() {
		for {
			// 模拟生成事件数据
			eventData := time.Now().Format("2006-01-02 15:04:05")
			// 将事件数据发送到事件流
			eventStream <- eventData

			// 等待1秒钟
			time.Sleep(1 * time.Second)
		}
	}()

	// 循环监听事件流，将事件数据写入响应流
	for {
		select {
		case eventData := <-eventStream:
			// 将事件数据写入响应流
			fmt.Fprintf(w, "data: %s\n\n", eventData)
			// 强制刷新响应流，确保数据被发送到客户端
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			// 客户端连接关闭，停止监听事件流
			return
		}
	}
}
